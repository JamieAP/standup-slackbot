package main

import (
	"log"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/ryanfaerman/fsm"
)

const (
	AreYouReady = "Hey, are you ready for standup? If you are say Yes or say No if you aren't taking part today."
)

func main() {
	slack := Slack{
		slack.New("xoxb-152520612096-sPKLUWO7FEYg0cMmPofGUyWt"),
		make(map[string]string, 0),
	}
	members, err := slack.GetChannelMembers("C4CSJ2XN3")
	if err != nil {
		log.Fatal(err)
	}

	membersQuestionnaires := make(map[string]StandupQuestionnaire, 0)
	for _, member := range members {
		standup := StandupQuestionnaire{
			State: "pending",
			Machine: &fsm.Machine{
				Rules:   &rules,
				Subject: nil,
			},
		}
		// I'd like to think I'm just using the library wrong with the chicken & egg problem here,
		// but this is how the docs suggest it be used... TODO unfuck
		standup.Machine.Subject = standup
		membersQuestionnaires[member] = standup
	}

	for member, standupQuestions := range membersQuestionnaires {
		go func() {
			for {
				switch standupQuestions.CurrentState() {
				case "ready?":
					ReadyState(member, standupQuestions, slack)
				}
			}
		}()
	}
}

func ReadyState(member string, standup StandupQuestionnaire, slack Slack) {
	ready, err := AskIfMemberReady(slack, member)
	if err != nil {
		log.Printf("Error asking member %s if they are ready: %v", member, err)
	}
	if ready == false {
		if err := standup.Machine.Transition("complete"); err != nil {
			log.Printf("Error transitioning state: %v", err)
		}
		return
	}
	if err := standup.Machine.Transition("yesterday?"); err != nil {
		log.Printf("Error transitioning state: %v", err)
	}
}

func AskIfMemberReady(slack Slack, member string) (bool, error) {
	msgSentAt := time.Now()
	questionRespChan := slack.AskQuestion(member, AreYouReady)
	readyResp := make(chan bool, 0)
	go func() {
		for {
			select {
			case resp := <-questionRespChan:
				respTime, err := time.Parse(time.RFC3339, resp.msg.Timestamp)
				if err != nil {
					log.Printf("Error parsing message timestamp: %v", err)
				}
				if respTime.After(msgSentAt) {
					msg := strings.ToLower(resp.msg.Text)
					if msg == "yes" {
						readyResp <- true
						return
					}
					if msg == "no" {
						readyResp <- false
						return
					}
				}

			case <-time.After(1 * time.Hour):
				readyResp <- false
				return
			}
		}
	}()
	return <-readyResp, nil
}
