FROM temporalio/admin-tools:latest

COPY setup.sh /setup.sh
RUN chmod +x /setup.sh

ENTRYPOINT ["/bin/sh", "/setup.sh"]
