package main

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/nlopes/slack"
	"github.com/ryanfaerman/fsm"
)

const (
	AreYouReady = "Hey :) are you ready for standup? If you are say Yes. If you aren't taking part today say No."
)

func main() {
	slack := Slack{
		slack.New("xoxb-152520612096-sPKLUWO7FEYg0cMmPofGUyWt"),
		make(map[string]string, 0),
		make(map[string]func(event *slack.MessageEvent), 0),
		sync.Mutex{},
	}
	go slack.StartRealTimeMessagingListener()
	members, err := slack.GetChannelMembers("C4PJYJEPM")
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
	resp := slack.AskQuestion(member, AreYouReady)
	spew.Dump(resp)
	if resp.err != nil {
		return false, fmt.Errorf("Error asking question: %v", resp.err)
	}
	msg := strings.ToLower(resp.msg.Text)
	if msg == "yes" {
		return true, nil
	}
	if msg == "no" {
		return false, nil
	}
	// todo ask again
	return false, errors.New("Unrecognised response")
}
