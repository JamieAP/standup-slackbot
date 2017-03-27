FROM alpine

RUN apk add --no-cache ca-certificates

RUN mkdir /app

WORKDIR /app

ADD standup-slackbot standup-slackbot

CMD ./standup-slackbot
