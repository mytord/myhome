FROM eclipse-mosquitto

COPY config /mosquitto/config
COPY init/init.sh /init.sh
RUN chmod +x /init.sh

# Оставляем root, т.к. root будет назначать права и запускать mosquitto от правильного пользователя
ENTRYPOINT ["/init.sh"]