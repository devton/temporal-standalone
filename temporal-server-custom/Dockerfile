FROM temporalio/server:1.31.2

COPY --chown=temporal:temporal config_template.yaml /etc/temporal/config/config_template.yaml
RUN chmod 644 /etc/temporal/config/config_template.yaml
