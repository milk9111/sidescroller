package systems

import (
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/obj"
)

// InputSystem mirrors legacy input into ECS components.
type InputSystem struct {
	Input  *obj.Input
	Entity ecs.Entity
}

// NewInputSystem creates an InputSystem.
func NewInputSystem(input *obj.Input, entity ecs.Entity) *InputSystem {
	return &InputSystem{Input: input, Entity: entity}
}

// Update copies input state into ECS.
func (s *InputSystem) Update(w *ecs.World) {
	if w == nil || s == nil || s.Input == nil || s.Entity.ID == 0 {
		return
	}
	st := w.GetInput(s.Entity)
	if st == nil {
		st = &components.InputState{}
		w.SetInput(s.Entity, st)
	}
	st.MoveX = s.Input.MoveX
	st.JumpPressed = s.Input.JumpPressed
	st.JumpHeld = s.Input.JumpHeld
	st.AimPressed = s.Input.AimPressed
	st.AimHeld = s.Input.AimHeld
	st.MouseLeftPressed = s.Input.MouseLeftPressed
	st.MouseWorldX = s.Input.MouseWorldX
	st.MouseWorldY = s.Input.MouseWorldY
	st.DashPressed = s.Input.DashPressed
	st.LastAimAngle = s.Input.LastAimAngle
	st.LastAimValid = s.Input.LastAimValid
}
