package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/script"
)

type ScriptSystem struct {
	runtime *script.Runtime
}

func NewScriptSystem() *ScriptSystem {
	return &ScriptSystem{runtime: script.NewRuntime()}
}

func (s *ScriptSystem) Update(w *ecs.World) {
	if s == nil || s.runtime == nil {
		return
	}
	s.runtime.Update(w)
}

func EmitEntitySignal(w *ecs.World, target ecs.Entity, source ecs.Entity, signalName string) bool {
	return script.EmitEntitySignal(w, target, source, signalName)
}
