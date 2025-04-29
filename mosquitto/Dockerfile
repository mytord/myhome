FROM eclipse-mosquitto

COPY config /mosquitto/config
COPY init/init.sh /init.sh
RUN chmod +x /init.sh && \
    chown -R mosquitto:mosquitto /mosquitto \
    chown -R mosquitto:mosquitto /mosquitto/config \
    chown -R mosquitto:mosquitto /mosquitto/log \
    chown -R mosquitto:mosquitto /mosquitto/data

USER mosquitto
ENTRYPOINT ["/init.sh"]