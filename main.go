package main

import (
	"log"
	"sync"
	"time"

	"github.com/nlopes/slack"
)

func main() {
	channel := "C4PJYJEPM"

	slackClient := &Slack{
		slack.New("xoxb-152520612096-sPKLUWO7FEYg0cMmPofGUyWt"),
		make(map[string]string),
		make(map[string]func(event *slack.MessageEvent)),
		sync.Mutex{},
	}

	members, err := slackClient.GetChannelMembers(channel)
	if err != nil {
		log.Fatalf("Error getting standup channel members: %v", err)
	}

	standup := NewStandup(slackClient, time.Now().Add(time.Hour * 1), members)

	log.Printf("%+v", standup.Start())
}
