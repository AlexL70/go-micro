# build a tiny docker image
FROM alpine:3.21

RUN mkdir /app

COPY mailApp /app

CMD [ "/app/mailApp" ]