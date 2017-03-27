package main

import "github.com/ryanfaerman/fsm"

const (
	AreYouReady          = "Hey :) are you ready for standup? If you are say Yes. If you aren't taking part today say No."
	NotUnderstoodYesOrNo = "Sorry, I didn't understand that, please say yes or no."
	Yesterday            = "Let's get started, what did you get done yesterday?"
	Today                = "Awesome, what are you working on today?"
	FinishedWhen         = "Great, when do you think you'll be finished with that?"
	Blockers             = "Is there anything blocking you or that could block you?"
	Done                 = "Thanks, have a great day!"
)

var (
	rules = fsm.Ruleset{}
)

func init() {
	rules.AddTransition(fsm.T{O: "pending", E: "ready?"})
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
