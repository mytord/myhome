FROM eclipse-mosquitto

# Устанавливаем gosu (в образе нет apk, поэтому через curl)
RUN apk add --no-cache curl && \
    curl -L https://github.com/tianon/gosu/releases/download/1.16/gosu-amd64 -o /usr/local/bin/gosu && \
    chmod +x /usr/local/bin/gosu

COPY config /mosquitto/config
COPY init/init.sh /init.sh
RUN chmod +x /init.sh

ENTRYPOINT ["/init.sh"]