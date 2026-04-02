package system

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type DebugMessageSystem struct{}

func NewDebugMessageSystem() *DebugMessageSystem {
	return &DebugMessageSystem{}
}

func (s *DebugMessageSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ent, ok := ecs.First(w, component.DebugMessageComponent.Kind())
	if !ok {
		return
	}

	debugMessage, ok := ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if !ok || debugMessage == nil || debugMessage.RemainingFrames <= 0 {
		return
	}

	debugMessage.RemainingFrames--
	if debugMessage.RemainingFrames > 0 {
		return
	}

	debugMessage.Message = ""
	if sprite, ok := ecs.Get(w, ent, component.SpriteComponent.Kind()); ok && sprite != nil {
		sprite.Disabled = true
		sprite.Image = nil
	}
}
