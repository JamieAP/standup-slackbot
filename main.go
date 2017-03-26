package main

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/nlopes/slack"
	"github.com/ryanfaerman/fsm"
)

const (
	AreYouReady          = "Hey :) are you ready for standup? If you are say Yes. If you aren't taking part today say No."
	NotUnderstoodYesOrNo = "Sorry, I didn't understand that, please say yes or no."
	Yesterday            = "Let's get started, what did you get done yesterday?"
	Today                = "Thanks, what are you working on today?"
	FinishedWhen         = "Great, when do you think you'll be finished with that?"
	Blockers             = "Is there anything blocking you or that could block you?"
	Done                 = "Thanks, have a great day!"
)

func main() {
	slack := &Slack{
		slack.New("xoxb-152520612096-sPKLUWO7FEYg0cMmPofGUyWt"),
		make(map[string]string),
		make(map[string]func(event *slack.MessageEvent)),
		sync.Mutex{},
	}
	go slack.StartRealTimeMessagingListener()
	members, err := slack.GetChannelMembers("C4PJYJEPM")
	if err != nil {
		log.Fatal(err)
	}

	membersQuestionnaires := make(map[string]*StandupQuestionnaire, 0)
	for _, member := range members {
		standup := &StandupQuestionnaire{State: "ready?"}
		machine := fsm.New(fsm.WithRules(rules), fsm.WithSubject(standup))
		standup.Machine = &machine
		membersQuestionnaires[member] = standup
	}

	for member, standupQuestions := range membersQuestionnaires {
		go func(member string, standup *StandupQuestionnaire) {
			for {
				switch standup.CurrentState() {
				case "ready?":
					ReadyState(member, standup, slack)
				case "yesterday?":
					YesterdayState(member, standup, slack)
				case "today?":
					TodaysState(member, standup, slack)
				case "finishedWhen?":
					FinishedWhenState(member, standup, slack)
				case "blockers?":
					BlockersState(member, standup, slack)
				case "complete":
					if err := slack.SendMessage(member, Done); err != nil {
						log.Printf("Error telling member %s standup is complete: %v", member, err)
					}
					return
				}
			}
		}(member, standupQuestions)
	}
}

func BlockersState(member string, standup *StandupQuestionnaire, slack *Slack) {
	resp := slack.AskQuestion(member, Blockers)
	if resp.err != nil {
		log.Printf("Error asking member %s about blockers: %v", member, resp.err)
		return
	}
	standup.SetBlockers(resp.msg.Text)
	if err := standup.Machine.Transition("complete"); err != nil {
		log.Printf("Error transitioning to complete: %v", err)
	}
}

func FinishedWhenState(member string, standup *StandupQuestionnaire, slack *Slack) {
	resp := slack.AskQuestion(member, FinishedWhen)
	if resp.err != nil {
		log.Printf("Error asking member %s when they'll be finished: %v", member, resp.err)
		return
	}
	standup.SetFinishedWhen(resp.msg.Text)
	if err := standup.Machine.Transition("blockers?"); err != nil {
		log.Printf("Error transitioning to blockers?: %v", err)
	}
}

func TodaysState(member string, standup *StandupQuestionnaire, slack *Slack) {
	resp := slack.AskQuestion(member, Today)
	if resp.err != nil {
		log.Printf("Error asking member %s about today: %v", member, resp.err)
		return
	}
	standup.SetTodaysUpdate(resp.msg.Text)
	if err := standup.Machine.Transition("finishedWhen?"); err != nil {
		log.Printf("Error transitioning to finishedWhen?: %v", err)
	}
}

func YesterdayState(member string, standup *StandupQuestionnaire, slack *Slack) {
	resp := slack.AskQuestion(member, Yesterday)
	if resp.err != nil {
		log.Printf("Error asking member %s about yesterday: %v", member, resp.err)
		return
	}
	standup.SetYesterdaysUpdate(resp.msg.Text)
	if err := standup.Machine.Transition("today?"); err != nil {
		log.Printf("Error transitioning to today?: %v", err)
	}
}

func ReadyState(member string, standup *StandupQuestionnaire, slack *Slack) {
	ready, err := AskIfMemberReady(slack, member)
	if err != nil {
		log.Printf("Error asking member %s if they are ready: %v", member, err)
		return
	}
	if ready == false {
		if err := standup.Machine.Transition("complete"); err != nil {
			log.Printf("Error transitioning to complete? from ready?: %v", err)
		}
		return
	}
	if err := standup.Machine.Transition("yesterday?"); err != nil {
		log.Printf("Error transitioning to yesterday?: %v", err)
	}
}

func AskIfMemberReady(slack *Slack, member string) (bool, error) {
	resp := slack.AskQuestion(member, AreYouReady)
	if resp.err != nil {
		return false, fmt.Errorf("Error asking question: %v", resp.err)
	}
	msg := strings.ToLower(resp.msg.Text)
	for msg != "yes" && msg != "no" {
		resp = slack.AskQuestion(member, NotUnderstoodYesOrNo)
		if resp.err != nil {
			return false, fmt.Errorf("Error asking question: %v", resp.err)
		}
		msg = resp.msg.Text
	}
	if msg == "yes" {
		return true, nil
	}
	return false, nil
}
