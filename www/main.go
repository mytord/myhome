package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	logger     = zap.Must(zap.NewDevelopment())
	chatID     = int64(254617095)
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	wwwServerRootUrl := os.Getenv("WWW_SERVER_ROOT_URL")
	webhookPath := "/tg/webhook"
	testPath := "/tg/test"
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
		SetPassword(os.Getenv("MQTT_PASSWORD")).
		SetClientID("ServerAppClient")
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
	mux.HandleFunc(testPath, testHandler)

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

	if update.CallbackQuery != nil {
		chatID := update.CallbackQuery.Message.Chat.ID
		data := update.CallbackQuery.Data
		user := update.CallbackQuery.From.UserName

		logger.Info("CallbackQuery от пользователя",
			zap.Int64("chat_id", chatID),
			zap.String("from", user),
			zap.String("data", data),
		)

		var payload map[string]interface{}

		switch data {
		case "pump_on":
			payload = map[string]interface{}{"command": "pump_on"}
		case "pump_on_60":
			payload = map[string]interface{}{"command": "pump_on", "minutes": 60}
		case "pump_on_120":
			payload = map[string]interface{}{"command": "pump_on", "minutes": 120}
		case "pump_off":
			payload = map[string]interface{}{"command": "pump_off"}
		case "valve_on_60":
			payload = map[string]interface{}{"command": "valve_on", "seconds": 60}
		case "valve_off":
			payload = map[string]interface{}{"command": "valve_off"}
		case "plant_interval_1":
			payload = map[string]interface{}{"command": "set_interval", "minutes": 1}
		case "plant_interval_5":
			payload = map[string]interface{}{"command": "set_interval", "minutes": 5}
		case "plant_interval_30":
			payload = map[string]interface{}{"command": "set_interval", "minutes": 30}
		case "status":
			payload = map[string]interface{}{"command": "status"}
		}

		ack := tgbotapi.NewCallback(update.CallbackQuery.ID, "✅ Команда принята: "+data)
		if _, err := bot.Request(ack); err != nil {
			logger.Error("Ошибка отправки Callback ответа", zap.Error(err))
		}

		if payload != nil {
			go func(data map[string]interface{}) {
				jsonPayload, err := json.Marshal(data)
				if err != nil {
					logger.Error("Ошибка сериализации JSON (callback)", zap.Error(err))
					return
				}
				token := mqttClient.Publish("commands", 0, false, jsonPayload)
				token.Wait()
				logger.Info("Команда из CallbackQuery отправлена в MQTT", zap.ByteString("payload", jsonPayload))
			}(payload)
		}

		w.WriteHeader(http.StatusOK)
		return
	}

	if update.Message == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	text := strings.TrimSpace(update.Message.Text)

	logger.Info("Получено новое сообщение",
		zap.Int64("chat_id", update.Message.Chat.ID),
		zap.String("from", update.Message.From.UserName),
		zap.String("text", text),
	)

	parts := strings.Fields(text)
	if len(parts) == 0 {
		w.WriteHeader(http.StatusOK)
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

		inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💧Вкл", "pump_on"),
				tgbotapi.NewInlineKeyboardButtonData("💧1 ч", "pump_on_60"),
				tgbotapi.NewInlineKeyboardButtonData("💧2 ч", "pump_on_120"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📊Статус", "status"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⛔Выкл", "pump_off"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🚰10L", "valve_on_60"),
				tgbotapi.NewInlineKeyboardButtonData("🚫🚰Выкл", "valve_off"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🪴1м", "plant_interval_1"),
				tgbotapi.NewInlineKeyboardButtonData("🪴5м", "plant_interval_5"),
				tgbotapi.NewInlineKeyboardButtonData("🪴30м", "plant_interval_30"),
			),
		)

		msg.ReplyMarkup = inlineKeyboard
		sentMsg, err := bot.Send(msg)
		if err != nil {
			logger.Error("Не могу отправить клавиатуру", zap.Error(err))
		} else {
			logger.Info("Клавиатура успешно отправлена", zap.Int("message_id", sentMsg.MessageID))
		}

		w.WriteHeader(http.StatusOK)
		return

	case "/pump_on":
		payload = map[string]interface{}{"command": "pump_on"}
		if arg != "" {
			if minutes, err := strconv.Atoi(arg); err == nil && minutes > 0 {
				payload["minutes"] = minutes
			}
		}

	case "/pump_off":
		payload = map[string]interface{}{"command": "pump_off"}

	case "/status":
		payload = map[string]interface{}{"command": "status"}

	case "/valve_on":
		payload = map[string]interface{}{"command": "valve_on"}
		if arg != "" {
			if seconds, err := strconv.Atoi(arg); err == nil && seconds > 0 {
				payload["seconds"] = seconds
			}
		}
	case "/valve_off":
		payload = map[string]interface{}{"command": "valve_off"}
	}

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

	w.WriteHeader(http.StatusOK)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("🔥 Test handler triggered")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK\n"))
}

func telegramMessageSender(mqttClient mqtt.Client) {
	if token := mqttClient.Subscribe("messages/#", 0, func(client mqtt.Client, msg mqtt.Message) {
		logger.Info("Есть сообщение на отправку в telegram.")

		text := string(msg.Payload())

		if strings.TrimSpace(text) == "" {
			logger.Warn("MQTT сообщение пустое")
			return
		}

		tgMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("[%s]\n%s", msg.Topic(), text))
		if _, err := bot.Send(tgMsg); err != nil {
			logger.Error("Ошибка отправки сообщения в Telegram", zap.Error(err))
		} else {
			logger.Info("Сообщение из MQTT отправлено в Telegram", zap.String("text", text))
		}
	}); token.Wait() && token.Error() != nil {
		logger.Fatal("Ошибка подписки на messages", zap.Error(token.Error()))
	}
}
