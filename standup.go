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
	machine *fsm.Machine
}

func (s *StandupQuestionnaire) CurrentState() fsm.State {
	return s.State
}

func (s *StandupQuestionnaire) SetState(state fsm.State) {
	s.State = state
}

func (s *StandupQuestionnaire) Apply(r *fsm.Ruleset) *fsm.Machine {
	if s.machine == nil {
		s.machine = &fsm.Machine{Subject: s}
	}

	s.machine.Rules = r
	return s.machine
}

