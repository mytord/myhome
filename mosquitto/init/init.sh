#!/bin/sh

PASSFILE="/mosquitto/config/passwordfile"

if [ ! -f "$PASSFILE" ]; then
  echo "Generating Mosquitto password file..."
  mosquitto_passwd -b -c "$PASSFILE" "${MQTT_USER}" "${MQTT_PASSWORD}"
else
  echo "Password file already exists."
fi

exec mosquitto -c /mosquitto/config/mosquitto.conf