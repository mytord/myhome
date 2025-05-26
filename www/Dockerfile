FROM golang:1.24-alpine

# Создаём рабочую директорию
WORKDIR /app

# Копируем go.mod и go.sum и ставим зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY ./www ./www

# Собираем бинарник
RUN go build -o www-app ./www/main.go

# Запускаем
CMD ["./www-app"]
