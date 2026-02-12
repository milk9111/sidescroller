package system

import (
	"image/color"
	"math"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AnchorSystem struct{}

func NewAnchorSystem() *AnchorSystem { return &AnchorSystem{} }

func (s *AnchorSystem) Update(w *ecs.World) {
	if w == nil || s == nil {
		return
	}

	// find player body
	playerEnt, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	playerBodyComp, ok := ecs.Get(w, playerEnt, component.PhysicsBodyComponent.Kind())
	if !ok || playerBodyComp.Body == nil {
		return
	}

	ecs.ForEach2(
		w,
		component.AnchorComponent.Kind(),
		component.AnchorTagComponent.Kind(),
		func(e ecs.Entity, aComp *component.Anchor, _ *component.AnchorTag) {
			// kinematic anchor: use transform for position and movement
			transform, ok := ecs.Get(w, e, component.TransformComponent.Kind())
			if !ok {
				return
			}

			if line, ok := ecs.Get(w, e, component.LineRenderComponent.Kind()); ok {
				pPos := playerBodyComp.Body.Position()
				line.StartX = pPos.X
				line.StartY = pPos.Y
				line.EndX = transform.X
				line.EndY = transform.Y
				if line.Width <= 0 {
					line.Width = 2
				}
				if line.Color == nil {
					line.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
				}
				if err := ecs.Add(w, e, component.LineRenderComponent.Kind(), line); err != nil {
					panic("anchor system: update line render: " + err.Error())
				}
			}

			// if no joint yet: drive the anchor transform toward its target.
			if _, has := ecs.Get(w, e, component.AnchorJointComponent.Kind()); !has {
				tx := aComp.TargetX
				ty := aComp.TargetY
				vx := tx - transform.X
				vy := ty - transform.Y
				d := math.Hypot(vx, vy)
				// threshold to consider the anchor "attached" at the hit point
				const attachThreshold = 6.0
				if d > attachThreshold {
					// move toward target without overshooting
					speed := aComp.Speed
					if speed <= 0 {
						speed = 12
					}
					step := speed
					if step > d {
						step = d
					}
					nx := vx / d
					ny := vy / d
					transform.X += nx * step
					transform.Y += ny * step
					if err := ecs.Add(w, e, component.TransformComponent.Kind(), transform); err != nil {
						panic("anchor system: update transform: " + err.Error())
					}

					return
				}

				transform.X = tx
				transform.Y = ty
				if err := ecs.Add(w, e, component.TransformComponent.Kind(), transform); err != nil {
					panic("anchor system: snap transform: " + err.Error())
				}

				// reached target: request slide joint to player anchored at the hit point
				pPos := playerBodyComp.Body.Position()
				dx := transform.X - pPos.X
				dy := transform.Y - pPos.Y
				dist := math.Hypot(dx, dy)
				// allow some slack so player can move left/right while grounded
				maxLen := math.Max(dist, 100000.0)
				req := &component.AnchorConstraintRequest{
					Mode:    component.AnchorConstraintSlide,
					AnchorX: transform.X,
					AnchorY: transform.Y,
					MinLen:  0,
					MaxLen:  maxLen,
					Applied: false,
				}
				if err := ecs.Add(w, e, component.AnchorConstraintRequestComponent.Kind(), req); err != nil {
					panic("anchor system: add constraint request: " + err.Error())
				}

				return
			}

			// joint exists: check player state to see if we should lock pivot
			jointComp, _ := ecs.Get(w, e, component.AnchorJointComponent.Kind())
			stateComp, ok := ecs.Get(w, playerEnt, component.PlayerStateMachineComponent.Kind())
			isFalling := false
			if ok && stateComp.State != nil && stateComp.State.Name() == "fall" {
				isFalling = true
			}
			if isFalling && jointComp.Slide != nil && jointComp.Pin == nil {
				// request pin joint to behave like a rope for smooth swinging
				px := aComp.TargetX
				py := aComp.TargetY
				req := &component.AnchorConstraintRequest{
					Mode:    component.AnchorConstraintPin,
					AnchorX: px,
					AnchorY: py,
					Applied: false,
				}
				if err := ecs.Add(w, e, component.AnchorConstraintRequestComponent.Kind(), req); err != nil {
					panic("anchor system: update constraint request: " + err.Error())
				}
			}
		})
}
