#!/bin/sh

PASSFILE="/mosquitto/config/passwordfile"

chown -R mosquitto:mosquitto /mosquitto/config
chown -R mosquitto:mosquitto /mosquitto/log
chown -R mosquitto:mosquitto /mosquitto/data

if [ ! -f "$PASSFILE" ]; then
  echo "Generating Mosquitto password file..."
  mosquitto_passwd -b -c "$PASSFILE" "${MQTT_USER}" "${MQTT_PASSWORD}"
else
  echo "Password file already exists."
fi

exec mosquitto -c /mosquitto/config/mosquitto.conf