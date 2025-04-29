FROM eclipse-mosquitto

COPY config /mosquitto/config
COPY init/init.sh /init.sh
RUN chmod +x /init.sh

ENTRYPOINT ["/init.sh"]