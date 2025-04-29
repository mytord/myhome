#!/bin/sh

set -e

# Создание директорий, если их нет
mkdir -p /mosquitto/config
mkdir -p /mosquitto/data
mkdir -p /mosquitto/log

# Назначение прав
chown -R mosquitto:mosquitto /mosquitto

# Генерация passwordfile (от имени mosquitto)
su mosquitto -c "mosquitto_passwd -b -c /mosquitto/config/passwordfile \"$MQTT_USER\" \"$MQTT_PASSWORD\""

# Запуск mosquitto (от имени mosquitto)
exec su mosquitto -c "mosquitto -c /mosquitto/config/mosquitto.conf"