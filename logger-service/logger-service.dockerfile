# build a tiny docker image
FROM alpine:3.21

RUN mkdir /app

COPY loggerApp /app

CMD [ "/app/loggerApp" ]