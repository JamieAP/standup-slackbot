# Standup Slackbot

A Slack bot for your standups.

If you scratch the surface, there is an extractable generic Slackbot framework somewhere in here.

### Usage

```
$ ./standup-slackbot --help

Usage: standup-slackbot [OPTIONS]

A slackbot for standups

Options:
  --slack-token=""              Slack API token ($SLACK_TOKEN)
  --standup-channel=""          The Slack channel to use for standups ($STANDUP_CHANNEL)
  --standup-time=""             The time standup should start in 24hr 00:00 format ($STANDUP_TIME)
  --standup-length-mins=60      The standup length time in minutes ($STANDUP_LENGTH_MINS)
  --time-zone="Europe/London"   The timezone IANA format e.g. Europe/London ($TIME_ZONE)
```

### What does it do?

At a set time every day it will DM everyone in the standup channel, 
ask them 4 questions and at the end post everyone's answers to the standup channel.

 ![example](http://j.mp/2nr3sqi)