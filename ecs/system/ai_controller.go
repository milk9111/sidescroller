package system

import (
	"image"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type AISystem struct {
	fsmCache map[string]*FSMDef
}

func NewAISystem() *AISystem {
	return &AISystem{
		fsmCache: map[string]*FSMDef{
			component.DefaultAIFSMName: DefaultEnemyFSM(),
		},
	}
}

func (e *AISystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	var playerPosX, playerPosY float64
	playerFound := false
	var playerEnt ecs.Entity
	if p, ok := ecs.First(w, component.PlayerTagComponent.Kind()); ok {
		playerEnt = p
		// Prefer physics body position so AI compares center-to-center coordinates.
		// Transform may represent top-left and can introduce side-biased ranges.
		if pb, ok := ecs.Get(w, playerEnt, component.PhysicsBodyComponent.Kind()); ok && pb.Body != nil {
			pos := pb.Body.Position()
			playerPosX = pos.X
			playerPosY = pos.Y
			playerFound = true
		}
	}

	ecs.ForEach8(w,
		component.AITagComponent.Kind(),
		component.AIComponent.Kind(),
		component.PhysicsBodyComponent.Kind(),
		component.AIStateComponent.Kind(),
		component.AIContextComponent.Kind(),
		component.AIConfigComponent.Kind(),
		component.AnimationComponent.Kind(),
		component.SpriteComponent.Kind(),
		func(ent ecs.Entity, _ *component.AITag, aiComp *component.AI, bodyComp *component.PhysicsBody, stateComp *component.AIState, ctxComp *component.AIContext, cfgComp *component.AIConfig, animComp *component.Animation, spriteComp *component.Sprite) {
			var fsm *FSMDef
			if cfgComp.Spec != nil {
				compiled, err := CompileFSMSpec(*cfgComp.Spec)
				if err == nil {
					fsm = compiled
				}
			} else {
				fsm = e.getFSM(cfgComp.FSM)
			}

			if fsm == nil {
				return
			}

			getPos := func() (x, y float64) {
				if bodyComp.Body != nil {
					pos := bodyComp.Body.Position()
					return pos.X, pos.Y
				}
				if t, ok := ecs.Get(w, ent, component.TransformComponent.Kind()); ok {
					return t.X, t.Y
				}
				return 0, 0
			}

			pendingEvents := make([]component.EventID, 0, 4)
			enqueue := func(ev component.EventID) {
				if ev == "" {
					return
				}
				pendingEvents = append(pendingEvents, ev)
			}

			// Consume any one-shot AI interrupt events (e.g. from combat)
			if irq, ok := ecs.Get(w, ent, component.AIStateInterruptComponent.Kind()); ok {
				if irq.Event != "" {
					enqueue(component.EventID(irq.Event))
				}
				_ = ecs.Remove(w, ent, component.AIStateInterruptComponent.Kind())
			}

			// Consume queued AI events (e.g. from phase enter or combat triggers).
			if q, ok := ecs.Get(w, ent, component.AIEventQueueComponent.Kind()); ok {
				for _, ev := range q.Events {
					if ev == "" {
						continue
					}
					enqueue(component.EventID(ev))
				}
				_ = ecs.Remove(w, ent, component.AIEventQueueComponent.Kind())
			}

			ctx := &AIActionContext{
				World:        w,
				Entity:       ent,
				AI:           aiComp,
				State:        stateComp,
				Context:      ctxComp,
				Config:       cfgComp,
				PlayerFound:  playerFound,
				PlayerX:      playerPosX,
				PlayerY:      playerPosY,
				PlayerEntity: playerEnt,
				GetPosition:  getPos,
				GetVelocity: func() (x, y float64) {
					if bodyComp.Body == nil {
						return 0, 0
					}

					vel := bodyComp.Body.Velocity()
					return vel.X, vel.Y
				},
				SetVelocity: func(x, y float64) {
					if bodyComp.Body == nil {
						return
					}

					bodyComp.Body.SetVelocityVector(cp.Vector{X: x, Y: y})
				},
				ChangeAnimation: func(animation string) {
					def, ok := animComp.Defs[animation]
					if !ok || animComp.Sheet == nil {
						return
					}
					animComp.Current = animation
					animComp.Frame = 0
					animComp.FrameTimer = 0
					animComp.Playing = true
					rect := image.Rect(def.ColStart*def.FrameW, def.Row*def.FrameH, def.ColStart*def.FrameW+def.FrameW, def.Row*def.FrameH+def.FrameH)
					spriteComp.Image = animComp.Sheet.SubImage(rect).(*ebiten.Image)
				},
				FacingLeft: func(facingLeft bool) {
					spriteComp.FacingLeft = facingLeft
				},
				EnqueueEvent: enqueue,
			}

			if stateComp.Current == "" {
				stateComp.Current = fsm.Initial
				applyActions(fsm.States[stateComp.Current].OnEnter, ctx)
			}

			enqueueSensorEvents(aiComp, playerFound, playerPosX, playerPosY, getPos, enqueue)

			// Run the state's While actions first so they can update context (e.g. timers)
			// and enqueue events that should be handled in the same tick.
			applyActions(fsm.States[stateComp.Current].While, ctx)

			// evaluate compiled transition checkers for the current state (after While)
			for _, ch := range fsm.Checkers {
				if ch.From != stateComp.Current {
					continue
				}
				if ch.Check != nil && ch.Check(ctx) {
					enqueue(ch.Event)
				}
			}

			// Process any pending events (from sensors, While actions, or checkers)
			processEvents(fsm, stateComp, ctx, pendingEvents)
		},
	)
}

func (e *AISystem) getFSM(name string) *FSMDef {
	if name == "" {
		name = component.DefaultAIFSMName
	}
	if e.fsmCache == nil {
		e.fsmCache = map[string]*FSMDef{}
	}
	if fsm, ok := e.fsmCache[name]; ok {
		return fsm
	}
	if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
		fsm, err := LoadFSMFromPrefab(name)
		if err == nil {
			e.fsmCache[name] = fsm
			return fsm
		}
		return nil
	}
	return e.fsmCache[component.DefaultAIFSMName]
}

func enqueueSensorEvents(ai *component.AI, playerFound bool, playerX, playerY float64, getPos func() (x, y float64), enqueue func(ev component.EventID)) {
	if ai == nil || enqueue == nil {
		return
	}
	if !playerFound {
		enqueue(component.EventID("loses_player"))
		return
	}
	ex, ey := getPos()
	dx := playerX - ex
	dy := playerY - ey

	// Use 2D Euclidean distance so vertical separation affects sensing.
	dist := math.Hypot(dx, dy)

	if ai.FollowRange > 0 {
		if dist <= ai.FollowRange {
			enqueue(component.EventID("sees_player"))
		} else {
			enqueue(component.EventID("loses_player"))
		}
	}
	if ai.AttackRange > 0 {
		// Enter attack a bit earlier than the nominal range so melee enemies
		// don't freeze in follow right outside attack distance.
		attackEnterRange := ai.AttackRange + 24
		// Use hysteresis to avoid rapid attack/follow toggling.
		attackExitRange := attackEnterRange + 10
		if dist <= attackEnterRange {
			enqueue(component.EventID("in_attack_range"))
		} else if dist > attackExitRange {
			enqueue(component.EventID("out_attack_range"))
		}
	}
}

func processEvents(fsm *FSMDef, state *component.AIState, ctx *AIActionContext, events []component.EventID) {
	if fsm == nil || state == nil || ctx == nil {
		return
	}
	for _, ev := range events {
		transitions, ok := fsm.Transitions[state.Current]
		if !ok {
			continue
		}
		next, ok := transitions[ev]
		if !ok || next == state.Current {
			continue
		}
		applyActions(fsm.States[state.Current].OnExit, ctx)
		state.Current = next
		applyActions(fsm.States[state.Current].OnEnter, ctx)
	}
}

func applyActions(actions []Action, ctx *AIActionContext) {
	for _, a := range actions {
		if a != nil {
			a(ctx)
		}
	}
}
