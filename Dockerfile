FROM alpine

RUN apk add --no-cache ca-certificates tzdata

RUN mkdir /app

WORKDIR /app

ADD standup-slackbot standup-slackbot

CMD ./standup-slackbot
