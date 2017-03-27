package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/ryanfaerman/fsm"
)

const (
	AreYouReady          = "Hey :) are you ready for standup? If you are say Yes. If you aren't taking part today say No."
	NotUnderstoodYesOrNo = "Sorry, I didn't understand that, please say yes or no."
	Yesterday            = "Let's get started, what did you get done since last standup?"
	Today                = "Awesome, what are you working on today?"
	FinishedWhen         = "Great, when do you think you'll be finished with that?"
	Blockers             = "Is there anything blocking you?"
	Done                 = "Thanks, have a great day!"
)

var (
	rules = fsm.Ruleset{}
)

func init() {
	rules.AddTransition(fsm.T{O: "ready?", E: "yesterday?"})
	rules.AddTransition(fsm.T{O: "ready?", E: "complete"})
	rules.AddTransition(fsm.T{O: "yesterday?", E: "today?"})
	rules.AddTransition(fsm.T{O: "today?", E: "finishedWhen?"})
	rules.AddTransition(fsm.T{O: "finishedWhen?", E: "blockers?"})
	rules.AddTransition(fsm.T{O: "blockers?", E: "complete"})
}

type StandupQuestionnaire struct {
	State        fsm.State
	Machine      *fsm.Machine
	Member       *slack.User
	yesterday    string
	today        string
	blockers     string
	finishedWhen string
}

func (s *StandupQuestionnaire) CurrentState() fsm.State {
	return s.State
}

func (s *StandupQuestionnaire) SetState(state fsm.State) {
	s.State = state
}

func (s *StandupQuestionnaire) SetYesterdaysUpdate(update string) {
	s.yesterday = update
}

func (s *StandupQuestionnaire) SetTodaysUpdate(update string) {
	s.today = update
}

func (s *StandupQuestionnaire) SetBlockers(update string) {
	s.blockers = update
}

func (s *StandupQuestionnaire) SetFinishedWhen(update string) {
	s.finishedWhen = update
}

func (s *StandupQuestionnaire) GetYesterdaysUpdate() string {
	return s.yesterday
}

func (s *StandupQuestionnaire) GetTodaysUpdate() string {
	return s.today
}

func (s *StandupQuestionnaire) GetBlockers() string {
	return s.blockers
}

func (s *StandupQuestionnaire) GetFinishedWhen() string {
	return s.finishedWhen
}

type Standup struct {
	Slack                       *Slack
	MemberStandupQuestionnaires map[string]*StandupQuestionnaire
	Context                     context.Context
	CancelFunc                  context.CancelFunc
}

func NewStandup(slack *Slack, finishTime time.Time, members map[string]*slack.User) Standup {
	membersQuestionnaires := make(map[string]*StandupQuestionnaire)
	for memberId, memberInfo := range members {
		if memberInfo.IsBot || memberInfo.Deleted {
			continue
		}
		questions := &StandupQuestionnaire{Member: memberInfo, State: "ready?"}
		machine := fsm.New(fsm.WithRules(rules), fsm.WithSubject(questions))
		questions.Machine = &machine
		membersQuestionnaires[memberId] = questions
	}
	ctx, cancelFunc := context.WithDeadline(context.Background(), finishTime)
	return Standup{
		MemberStandupQuestionnaires: membersQuestionnaires,
		Context:                     ctx,
		CancelFunc:                  cancelFunc,
		Slack:                       slack,
	}
}

func (s Standup) Start() map[string]*StandupQuestionnaire {
	go s.Slack.StartRealTimeMessagingListener(s.Context)
	for member, questions := range s.MemberStandupQuestionnaires {
		go func(member string, questions *StandupQuestionnaire) {
			for {
				if s.Context.Err() != nil {
					log.Printf("Context completed before member %s finished: %v", member, s.Context.Err())
					return
				}
				switch questions.CurrentState() {
				case "ready?":
					s.readyState(member, questions)
				case "yesterday?":
					s.yesterdayState(member, questions)
				case "today?":
					s.todayState(member, questions)
				case "finishedWhen?":
					s.finishedWhenState(member, questions)
				case "blockers?":
					s.blockersState(member, questions)
				case "complete":
					s.completeState(member)
					return
				}
			}
		}(member, questions)
	}
	s.waitForCompletion()
	return s.MemberStandupQuestionnaires
}

func (s Standup) waitForCompletion() {
	for {
		select {
		case <-s.Context.Done():
			return
		case <-time.After(1 * time.Minute):
			complete := 0
			for _, questionnaires := range s.MemberStandupQuestionnaires {
				if questionnaires.CurrentState() == "complete" {
					complete++
				}
			}
			if complete == len(s.MemberStandupQuestionnaires) {
				s.CancelFunc()
				return
			}
		}
	}
}

func (s Standup) completeState(member string) {
	if _, err := s.Slack.SendMessage(member, Done); err != nil {
		log.Printf("Error telling member %s standup is complete: %v", member, err)
	}
}

func (s Standup) blockersState(member string, standup *StandupQuestionnaire) {
	ctx, _ := context.WithCancel(s.Context)
	resp := s.Slack.AskQuestion(member, Blockers, ctx)
	if resp.err != nil {
		log.Printf("Error asking member %s about blockers: %v", member, resp.err)
		return
	}
	standup.SetBlockers(resp.msg.Text)
	if err := standup.Machine.Transition("complete"); err != nil {
		log.Printf("Error transitioning to complete: %v", err)
	}
}

func (s Standup) finishedWhenState(member string, standup *StandupQuestionnaire) {
	ctx, _ := context.WithCancel(s.Context)
	resp := s.Slack.AskQuestion(member, FinishedWhen, ctx)
	if resp.err != nil {
		log.Printf("Error asking member %s when they'll be finished: %v", member, resp.err)
		return
	}
	standup.SetFinishedWhen(resp.msg.Text)
	if err := standup.Machine.Transition("blockers?"); err != nil {
		log.Printf("Error transitioning to blockers?: %v", err)
	}
}

func (s Standup) todayState(member string, standup *StandupQuestionnaire) {
	ctx, _ := context.WithCancel(s.Context)
	resp := s.Slack.AskQuestion(member, Today, ctx)
	if resp.err != nil {
		log.Printf("Error asking member %s about today: %v", member, resp.err)
		return
	}
	standup.SetTodaysUpdate(resp.msg.Text)
	if err := standup.Machine.Transition("finishedWhen?"); err != nil {
		log.Printf("Error transitioning to finishedWhen?: %v", err)
	}
}

func (s Standup) yesterdayState(member string, standup *StandupQuestionnaire) {
	ctx, _ := context.WithCancel(s.Context)
	resp := s.Slack.AskQuestion(member, Yesterday, ctx)
	if resp.err != nil {
		log.Printf("Error asking member %s about yesterday: %v", member, resp.err)
		return
	}
	standup.SetYesterdaysUpdate(resp.msg.Text)
	if err := standup.Machine.Transition("today?"); err != nil {
		log.Printf("Error transitioning to today?: %v", err)
	}
}

func (s Standup) readyState(member string, standup *StandupQuestionnaire) {
	ready, err := s.AskIfMemberReady(member)
	if err != nil {
		log.Printf("Error asking member %s if they are ready: %v", member, err)
		return
	}
	if ready == false {
		if err := standup.Machine.Transition("complete"); err != nil {
			log.Printf("Error transitioning to complete?: %v", err)
		}
		return
	}
	if err := standup.Machine.Transition("yesterday?"); err != nil {
		log.Printf("Error transitioning to yesterday?: %v", err)
	}
}

func (s Standup) AskIfMemberReady(member string) (bool, error) {
	ctx, _ := context.WithCancel(s.Context)
	resp := s.Slack.AskQuestion(member, AreYouReady, ctx)
	if resp.err != nil {
		return false, fmt.Errorf("Error asking question: %v", resp.err)
	}
	msg := strings.ToLower(resp.msg.Text)
	for msg != "yes" && msg != "no" {
		ctx, _ := context.WithCancel(ctx)
		resp = s.Slack.AskQuestion(member, NotUnderstoodYesOrNo, ctx)
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
