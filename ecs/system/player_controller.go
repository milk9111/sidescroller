package system

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	playerMoveSpeed = 260.0
	playerJumpSpeed = 600.0
	groundedEpsilon = 1.0
)

type PlayerControllerSystem struct{}

func NewPlayerControllerSystem() *PlayerControllerSystem {
	return &PlayerControllerSystem{}
}

func (p *PlayerControllerSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	entities := w.Query(
		component.PlayerTagComponent.Kind(),
		component.InputComponent.Kind(),
		component.PhysicsBodyComponent.Kind(),
	)
	for _, e := range entities {
		input, ok := ecs.Get(w, e, component.InputComponent)
		if !ok {
			continue
		}
		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok || bodyComp.Body == nil {
			continue
		}

		vel := bodyComp.Body.Velocity()
		vel.X = input.MoveX * playerMoveSpeed

		if input.JumpPressed && math.Abs(vel.Y) < groundedEpsilon {
			vel.Y = -playerJumpSpeed
		}

		bodyComp.Body.SetVelocityVector(vel)
		bodyComp.Body.SetAngle(0)
		bodyComp.Body.SetAngularVelocity(0)
	}
}
