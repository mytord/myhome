#!/bin/sh

set -e

# Создание директорий
mkdir -p /mosquitto/config /mosquitto/data /mosquitto/log

# Назначение прав
chown -R mosquitto:mosquitto /mosquitto

# Генерация passwordfile от имени mosquitto
gosu mosquitto mosquitto_passwd -b -c /mosquitto/config/passwordfile "$MQTT_USER" "$MQTT_PASSWORD"

# Запуск Mosquitto от имени mosquitto
exec gosu mosquitto mosquitto -c /mosquitto/config/mosquitto.conf