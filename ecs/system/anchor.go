package system

import (
	"image/color"
	"math"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AnchorSystem struct{}

func NewAnchorSystem() *AnchorSystem { return &AnchorSystem{} }

func (s *AnchorSystem) Update(w *ecs.World) {
	if w == nil || s == nil {
		return
	}

	playerEnt, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	playerBodyComp, ok := ecs.Get(w, playerEnt, component.PhysicsBodyComponent.Kind())
	if !ok || playerBodyComp.Body == nil {
		return
	}

	playerComp, ok := ecs.Get(w, playerEnt, component.PlayerComponent.Kind())
	if !ok || playerComp == nil {
		return
	}
	reelSpeed := math.Max(0, playerComp.AnchorReelSpeed)
	maxAnchorLength := float64(common.AnchorMaxDistance)

	input, ok := ecs.Get(w, playerEnt, component.InputComponent.Kind())
	if !ok {
		return
	}
	_ = input

	ecs.ForEach2(
		w,
		component.AnchorComponent.Kind(),
		component.AnchorTagComponent.Kind(),
		func(e ecs.Entity, aComp *component.Anchor, _ *component.AnchorTag) {
			if input.AnchorReleasePressed && !ecs.Has(w, e, component.AnchorPendingDestroyComponent.Kind()) {
				_ = ecs.Add(w, e, component.AnchorPendingDestroyComponent.Kind(), &component.AnchorPendingDestroy{})
				return
			}

			playerCollision, _ := ecs.Get(w, playerEnt, component.PlayerCollisionComponent.Kind())
			stateComp, _ := ecs.Get(w, playerEnt, component.PlayerStateMachineComponent.Kind())
			isSwinging := stateComp != nil && stateComp.State != nil && stateComp.State.Name() == "swing"
			v := playerBodyComp.Body.Velocity()
			isFalling := v.Y > 0
			inputComp, _ := ecs.Get(w, playerEnt, component.InputComponent.Kind())
			isReeling := inputComp != nil && inputComp.AnchorReelIn
			isReelingOut := inputComp != nil && inputComp.AnchorReelOut
			if isReeling && isReelingOut {
				isReeling = false
				isReelingOut = false
			}
			desiredMode := component.AnchorConstraintSlide
			isGrounded := playerCollision != nil && (playerCollision.Grounded || playerCollision.GroundGrace > 0)
			// Reel-out needs slide mode so the rope can lengthen without pushing the
			// player to an exact radius. Otherwise keep the existing pin behavior for
			// active swing/tight-rope states.
			if isReeling {
				desiredMode = component.AnchorConstraintPin
			} else if isSwinging || (!isGrounded && isFalling) {
				desiredMode = component.AnchorConstraintPin
			}

			pPos := playerBodyComp.Body.Position()
			dx := aComp.TargetX - pPos.X
			dy := aComp.TargetY - pPos.Y
			dist := math.Hypot(dx, dy)
			minLen := math.Max(0, playerComp.AnchorMinLength)
			targetLen := dist
			if isReeling {
				targetLen = math.Max(minLen, dist-reelSpeed)
			} else if isReelingOut {
				targetLen = math.Min(maxAnchorLength, dist+reelSpeed)
			}
			// Default: do not allow extension beyond current distance unless
			// the player is grounded and actively moving. This prevents the
			// rope from extending while on walls or standing still on ground.
			maxLen := math.Max(minLen, targetLen)
			// check player input to see if player is moving or jumping while grounded
			moving := inputComp != nil && inputComp.MoveX != 0
			jumping := inputComp != nil && (inputComp.JumpPressed || inputComp.Jump)
			allowExtend := isGrounded && !isSwinging && (moving || jumping)
			if allowExtend && !isReeling && !isReelingOut {
				maxLen = math.Max(dist, 100000.0)
			} else if isReelingOut {
				maxLen = math.Min(maxAnchorLength, math.Max(minLen, targetLen))
			}
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

				// reached target: request the desired constraint mode.
				req := &component.AnchorConstraintRequest{
					TargetEntity: uint64(playerEnt),
					Mode:         desiredMode,
					AnchorX:      transform.X,
					AnchorY:      transform.Y,
					MinLen:       minLen,
					MaxLen:       maxLen,
					Applied:      false,
				}
				if err := ecs.Add(w, e, component.AnchorConstraintRequestComponent.Kind(), req); err != nil {
					panic("anchor system: add constraint request: " + err.Error())
				}

				return
			}

			// Joint exists: keep switching mode based on grounded/wall contact.
			jointComp, _ := ecs.Get(w, e, component.AnchorJointComponent.Kind())
			if jointComp == nil {
				return
			}

			alreadyDesired := (desiredMode == component.AnchorConstraintSlide && jointComp.Slide != nil && jointComp.Pin == nil && jointComp.Pivot == nil) ||
				(desiredMode == component.AnchorConstraintPin && jointComp.Pin != nil && jointComp.Slide == nil && jointComp.Pivot == nil)
			// Keep processing while in slide mode so max rope length can react to
			// grounded movement/jump intent every frame. Returning early here leaves
			// a stale short slide joint that can yank the player upward when moving
			// away from the anchor on ground.
			if alreadyDesired && desiredMode != component.AnchorConstraintSlide && !isReeling && !isReelingOut {
				return
			}

			// For existing joints, only allow extension when player is grounded
			// and moving or jumping. Recompute maxLen similarly to the initial attach logic.
			moving = inputComp != nil && inputComp.MoveX != 0
			jumping = inputComp != nil && (inputComp.JumpPressed || inputComp.Jump)
			allowExtend = isGrounded && !isSwinging && (moving || jumping)
			targetLen = dist
			if isReeling {
				targetLen = math.Max(minLen, dist-reelSpeed)
			} else if isReelingOut {
				targetLen = math.Min(maxAnchorLength, dist+reelSpeed)
			}
			if allowExtend && !isReeling && !isReelingOut {
				maxLen = math.Max(dist, 100000.0)
			} else {
				maxLen = math.Max(minLen, targetLen)
				if isReelingOut {
					maxLen = math.Min(maxAnchorLength, maxLen)
				}
			}

			req := &component.AnchorConstraintRequest{
				TargetEntity: uint64(playerEnt),
				Mode:         desiredMode,
				AnchorX:      aComp.TargetX,
				AnchorY:      aComp.TargetY,
				MinLen:       minLen,
				MaxLen:       maxLen,
				Applied:      false,
			}
			if err := ecs.Add(w, e, component.AnchorConstraintRequestComponent.Kind(), req); err != nil {
				panic("anchor system: update constraint request: " + err.Error())
			}
		})
}
