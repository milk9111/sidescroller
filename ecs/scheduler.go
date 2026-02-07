package ecs

type System interface {
	Update(w *World)
}

type Scheduler struct {
	systems []System
}

func NewScheduler(systems ...System) *Scheduler {
	copied := append([]System(nil), systems...)
	return &Scheduler{systems: copied}
}

func (s *Scheduler) Add(system System) {
	if system == nil {
		return
	}
	s.systems = append(s.systems, system)
}

func (s *Scheduler) Update(w *World) {
	for _, system := range s.systems {
		system.Update(w)
	}
}

func (s *Scheduler) Systems() []System {
	systems := make([]System, 0, len(s.systems))
	return append(systems, s.systems...)
}
