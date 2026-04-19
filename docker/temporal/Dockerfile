FROM temporalio/auto-setup:1.24.3

USER root
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /railway-entrypoint.sh
RUN chmod +x /railway-entrypoint.sh

COPY config/dynamicconfig/docker.yaml /etc/temporal/config/dynamicconfig/docker.yaml

USER temporal
ENTRYPOINT ["/railway-entrypoint.sh"]
