package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var (
	bot        *tgbotapi.BotAPI
	mqttClient mqtt.Client
	logger     = zap.Must(zap.NewProduction())
	chatID     = int64(254617095)
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	wwwServerRootUrl := os.Getenv("WWW_SERVER_ROOT_URL")
	webhookPath := "/tg/webhook"
	webhookUrl := wwwServerRootUrl + webhookPath

	if botToken == "" {
		log.Fatal("Не заданы обязательные переменные окружения")
	}

	// Telegram Bot
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Fatal("Ошибка инициализации бота:", zap.Error(err))
	}

	// Установка Webhook
	webhookConfig, err := tgbotapi.NewWebhook(webhookUrl)
	if err != nil {
		logger.Fatal("Ошибка webhook:", zap.Error(err))
	}
	_, err = bot.Request(webhookConfig)
	if err != nil {
		logger.Fatal("Ошибка запроса webhook:", zap.Error(err))
	}

	// MQTT
	opts := mqtt.NewClientOptions().
		AddBroker(os.Getenv("MQTT_BROKER")).
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second).
		SetUsername(os.Getenv("MQTT_USER")).
		SetPassword(os.Getenv("MQTT_PASSWORD"))
	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Fatal("Ошибка подключения к MQTT", zap.Error(token.Error()))
	}
	defer mqttClient.Disconnect(250)

	// MQTT подписка
	telegramMessageSender(mqttClient)

	// HTTP сервер
	mux := http.NewServeMux()
	mux.HandleFunc(webhookPath, telegramCommandHandler)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		logger.Info("HTTP сервер слушает :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Ошибка HTTP сервера", zap.Error(err))
		}
	}()

	// Ждём сигнал завершения
	<-ctx.Done()
	logger.Info("Остановка приложения...")

	// Контекст для graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Завершаем HTTP сервер
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Ошибка при остановке HTTP сервера", zap.Error(err))
	}

	logger.Info("Завершено")
}

func telegramCommandHandler(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if update.Message == nil {
		return
	}

	text := strings.TrimSpace(update.Message.Text)

	logger.Info("Получено новое сообщение",
		zap.Int64("from", update.Message.Chat.ID),
		zap.String("from", update.Message.From.UserName),
		zap.String("text", text),
	)

	parts := strings.Fields(text) // разбиваем на слова
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	var payload map[string]interface{}

	switch cmd {
	case "/start":
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите команду:")
		keyboard := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/pump_on"),
				tgbotapi.NewKeyboardButton("/pump_on 60"),
				tgbotapi.NewKeyboardButton("/pump_on 120"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/status"),
			),
			tgbotapi.NewKeyboardButtonRow(
				tgbotapi.NewKeyboardButton("/pump_off"),
			),
		)
		msg.ReplyMarkup = keyboard
		_, err := bot.Send(msg)
		if err != nil {
			logger.Info("Не могу отправить клавиатуру")
		}
	case "/pump_on":
		payload = map[string]interface{}{"command": "pump_on"}

		// Попробуем разобрать аргумент как продолжительность
		if arg != "" {
			if minutes, err := strconv.Atoi(arg); err == nil && minutes > 0 {
				payload["minutes"] = minutes
			}
		}

	case "/pump_off":
		payload = map[string]interface{}{"command": "pump_off"}

	case "/status":
		payload = map[string]interface{}{"command": "status"}

	default:
	}

	// Публикация в MQTT — асинхронно
	if payload != nil {
		go func(data map[string]interface{}) {
			jsonPayload, err := json.Marshal(data)
			if err != nil {
				logger.Error("Ошибка сериализации JSON", zap.Error(err))
				return
			}
			token := mqttClient.Publish("commands", 0, false, jsonPayload)
			token.Wait()
			logger.Info("Команда опубликована в MQTT", zap.ByteString("payload", jsonPayload))
		}(payload)
	}
}

func telegramMessageSender(mqttClient mqtt.Client) {
	if token := mqttClient.Subscribe("messages", 0, func(client mqtt.Client, msg mqtt.Message) {
		logger.Info("Есть сообщение на отправку в telegram.")

		text := string(msg.Payload())

		if strings.TrimSpace(text) == "" {
			logger.Warn("MQTT сообщение пустое")
			return
		}

		tgMsg := tgbotapi.NewMessage(chatID, text)
		if _, err := bot.Send(tgMsg); err != nil {
			logger.Error("Ошибка отправки сообщения в Telegram", zap.Error(err))
		} else {
			logger.Info("Сообщение из MQTT отправлено в Telegram", zap.String("text", text))
		}
	}); token.Wait() && token.Error() != nil {
		logger.Fatal("Ошибка подписки на messages", zap.Error(token.Error()))
	}
}
