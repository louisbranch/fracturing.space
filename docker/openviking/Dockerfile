FROM ghcr.io/volcengine/openviking:v0.2.10

COPY docker/openviking/port_forward.py /app/port_forward.py
COPY docker/openviking/entrypoint.sh /app/entrypoint.sh

RUN chmod 0755 /app/entrypoint.sh /app/port_forward.py

ENTRYPOINT ["/app/entrypoint.sh"]
