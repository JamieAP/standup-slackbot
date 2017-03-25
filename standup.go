package main

import "github.com/ryanfaerman/fsm"

var (
	rules = fsm.Ruleset{}
)

func init() {
	rules.AddTransition(fsm.T{"pending", "ready?"})
	rules.AddTransition(fsm.T{"ready?", "yesterday?"})
	rules.AddTransition(fsm.T{"ready?", "complete"})
	rules.AddTransition(fsm.T{"yesterday?", "today?"})
	rules.AddTransition(fsm.T{"today?", "finishedWhen?"})
	rules.AddTransition(fsm.T{"finishedWhen?", "blockers?"})
	rules.AddTransition(fsm.T{"blockers?", "complete"})
}

type StandupQuestionnaire struct {
	State   fsm.State
	Machine *fsm.Machine
}

func (s StandupQuestionnaire) CurrentState() fsm.State {
	return s.State
}

func (s StandupQuestionnaire) SetState(state fsm.State) {
	s.State = state
}
