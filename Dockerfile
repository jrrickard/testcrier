FROM alpine:latest
RUN mkdir -p /opt/bot
ADD dist/testcrier-linux /opt/bot/testcrier
USER nobody
EXPOSE 10000 
ENTRYPOINT ["/opt/bot/testcrier"]
