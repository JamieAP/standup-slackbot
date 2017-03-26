package main

import "github.com/ryanfaerman/fsm"

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
