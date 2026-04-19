FROM casbin/casdoor:v1.693.0

# Install curl for healthcheck and API calls
USER root
RUN apk add --no-cache curl jq postgresql-client

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER 1000
ENTRYPOINT ["/entrypoint.sh"]
