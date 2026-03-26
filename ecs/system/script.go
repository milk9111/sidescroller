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

func EmitEntitySignalWithPosition(w *ecs.World, target ecs.Entity, source ecs.Entity, signalName string, positionX, positionY float64, hasPosition bool) bool {
	return script.EmitEntitySignalWithPosition(w, target, source, signalName, positionX, positionY, hasPosition)
}

func BroadcastSignalWithPosition(w *ecs.World, source ecs.Entity, signalName string, positionX, positionY float64, hasPosition bool, excludeTargets ...ecs.Entity) int {
	return script.BroadcastSignalWithPosition(w, source, signalName, positionX, positionY, hasPosition, excludeTargets...)
}

func ClearGlobalHitSignalQueue(w *ecs.World) {
	script.ClearGlobalHitSignalQueue(w)
}

func QueueGlobalHitSignalWithPosition(w *ecs.World, source ecs.Entity, excludeTarget ecs.Entity, positionX, positionY float64, hasPosition bool) bool {
	return script.QueueGlobalHitSignalWithPosition(w, source, excludeTarget, positionX, positionY, hasPosition)
}
