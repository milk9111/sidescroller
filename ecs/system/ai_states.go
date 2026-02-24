package system

import (
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	entitypkg "github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/prefabs"
)

type Action func(ctx *AIActionContext)

type AIActionContext struct {
	World           *ecs.World
	Entity          ecs.Entity
	AI              *component.AI
	State           *component.AIState
	Context         *component.AIContext
	Config          *component.AIConfig
	PlayerFound     bool
	PlayerX         float64
	PlayerY         float64
	PlayerEntity    ecs.Entity
	GetPosition     func() (x, y float64)
	GetVelocity     func() (x, y float64)
	SetVelocity     func(x, y float64)
	ChangeAnimation func(name string)
	FacingLeft      func(facingLeft bool)
	EnqueueEvent    func(ev component.EventID)
}

type StateDef struct {
	OnEnter []Action
	While   []Action
	OnExit  []Action
}

type FSMDef struct {
	Initial     component.StateID
	States      map[component.StateID]StateDef
	Transitions map[component.StateID]map[component.EventID]component.StateID
	Checkers    []TransitionCheckerDef
}

var actionRegistry = map[string]func(any) Action{
	"print": func(arg any) Action {
		msg := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			fmt.Println("ai:", msg)
		}
	},
	"set_animation": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			if ctx != nil && ctx.ChangeAnimation != nil {
				ctx.ChangeAnimation(name)
			}
		}
	},
	"play_audio": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			audioComp, ok := ecs.Get(ctx.World, ctx.Entity, component.AudioComponent.Kind())
			if !ok {
				return
			}

			for i, audioName := range audioComp.Names {
				if audioName != name {
					continue
				}

				audioComp.Play[i] = true
			}
		}
	},
	"stop_audio": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			audioComp, ok := ecs.Get(ctx.World, ctx.Entity, component.AudioComponent.Kind())
			if !ok {
				return
			}

			for i, audioName := range audioComp.Names {
				if audioName != name {
					continue
				}

				audioComp.Stop[i] = true
			}
		}
	},
	"stop_x": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			_, y := ctx.GetVelocity()
			ctx.SetVelocity(0, y)
		}
	},
	"stop_xy": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.SetVelocity == nil {
				return
			}
			ctx.SetVelocity(0, 0)
		}
	},
	"jump": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			x, _ := ctx.GetVelocity()
			// determine height from arg: accept numeric or map{"height": val}
			h := 0.0
			switch v := arg.(type) {
			case float64:
				h = v
			case float32:
				h = float64(v)
			case int:
				h = float64(v)
			case map[string]any:
				if vv, ok := numberFromMap(v, "height"); ok {
					h = vv
				}
			}
			if h <= 0 {
				// sensible default jump impulse
				h = 160
			}
			ctx.SetVelocity(x, -h)
		}
	},
	"move_towards_player": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.AI == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}
			ex, ey := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dy := ctx.PlayerY - ey

			// Stop slightly before nominal attack range so we don't overshoot the
			// target before the FSM transitions into attack.
			stopDistance := ctx.AI.AttackRange
			if stopDistance < 24 {
				stopDistance = 24
			}

			// Reduce jitter when vertically stacked with the player by using a
			// wider horizontal deadzone.
			horizontalDeadzone := 4.0
			if math.Abs(dy) > 24 {
				horizontalDeadzone = 10
			}

			dir := 0.0
			if math.Abs(dx) > horizontalDeadzone && math.Abs(dx) > stopDistance {
				if dx > 0 {
					dir = 1
				} else {
					dir = -1
				}
			}

			// Separation steering: compute a small repulsive vector from nearby
			// AI neighbors so enemies spread instead of clustering. Prefer
			// physics body positions when available for more accurate centroids.
			if ctx.World != nil {
				const desiredSeparation = 40.0
				const verticalNeighborBand = 40.0
				const maxRepel = 1.0
				const repelWeight = 0.9

				repelX := 0.0
				repelY := 0.0

				ecs.ForEach3(ctx.World,
					component.AITagComponent.Kind(),
					component.PhysicsBodyComponent.Kind(),
					component.TransformComponent.Kind(),
					func(other ecs.Entity, _ *component.AITag, ob *component.PhysicsBody, ot *component.Transform) {
						if other == ctx.Entity {
							return
						}

						// Determine neighbor position (prefer physics body)
						nx, ny := 0.0, 0.0
						if ob != nil && ob.Body != nil {
							p := ob.Body.Position()
							nx, ny = p.X, p.Y
						} else if ot != nil {
							nx, ny = ot.X, ot.Y
						} else {
							return
						}

						// Only consider neighbors roughly on the same platform level
						if math.Abs(ny-ey) > verticalNeighborBand {
							return
						}

						dx := ex - nx
						dy := ey - ny
						dist := math.Hypot(dx, dy)
						if dist < 0.001 || dist >= desiredSeparation {
							return
						}

						// stronger push when very close, smooth to zero at desiredSeparation
						strength := (desiredSeparation - dist) / desiredSeparation
						// normalized direction from neighbor to self
						nxDir := dx / dist
						nyDir := dy / dist
						repelX += nxDir * strength
						repelY += nyDir * strength
					},
				)

				// apply only horizontal component to movement, cap magnitude
				mag := math.Hypot(repelX, repelY)
				if mag > 0.0001 {
					if mag > maxRepel {
						repelX = (repelX / mag) * maxRepel
					}
					// apply horizontal influence scaled by weight
					dir += repelX * repelWeight
					if dir > 1 {
						dir = 1
					} else if dir < -1 {
						dir = -1
					}
					if math.Abs(dir) < 0.15 {
						dir = 0
					}
				}
			}

			// If moving horizontally, check for ground ahead on the current platform.
			// If no ground is found, stop moving to avoid falling off edges.
			// Consult precomputed navigation data set by the AINavigationSystem.
			if ctx.World != nil && dir != 0 {
				if nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind()); ok && nav != nil {
					if dir > 0 && !nav.GroundAheadRight {
						dir = 0
					} else if dir < 0 && !nav.GroundAheadLeft {
						dir = 0
					}
				}
			}

			_, y := ctx.GetVelocity()
			ctx.SetVelocity(dir*ctx.AI.MoveSpeed, y)
		}
	},
	"move_towards_player_2d": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.AI == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.SetVelocity == nil {
				return
			}

			ex, ey := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dy := ctx.PlayerY - ey
			dist := math.Hypot(dx, dy)
			if dist < 1e-4 {
				ctx.SetVelocity(0, 0)
				return
			}

			stopDistance := ctx.AI.AttackRange + 20
			if stopDistance < 20 {
				stopDistance = 20
			}
			if dist <= stopDistance {
				ctx.SetVelocity(0, 0)
				return
			}

			nx := dx / dist
			ny := dy / dist
			ctx.SetVelocity(nx*ctx.AI.MoveSpeed, ny*ctx.AI.MoveSpeed)
		}
	},
	"face_player": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || !ctx.PlayerFound || ctx.GetPosition == nil || ctx.FacingLeft == nil {
				return
			}
			ex, _ := ctx.GetPosition()
			ctx.FacingLeft(ctx.PlayerX < ex)
		}
	},
	"start_timer": func(arg any) Action {
		seconds := asFloat(arg)
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil {
				return
			}
			fmt.Println("starting timer for", seconds, "seconds")
			ctx.Context.Timer = seconds
		}
	},
	"start_attack_timer": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil || ctx.AI == nil {
				return
			}
			frames := float64(ctx.AI.AttackFrames)
			if frames <= 0 {
				frames = 20
			}
			// store timer in seconds (frames / TPS) to match tick_timer which
			// decrements by 1/ebiten.ActualTPS() each update
			tps := ebiten.ActualTPS()
			if tps <= 0 {
				ctx.Context.Timer = frames
			} else {
				ctx.Context.Timer = frames / tps
			}
		}
	},
	"start_cooldown": func(arg any) Action {
		// Accept a numeric frames argument (int/float) or a map with "frames".
		frames := 0
		if arg != nil {
			switch v := arg.(type) {
			case int:
				frames = v
			case float64:
				frames = int(v)
			case map[string]any:
				if fv, ok := v["frames"].(float64); ok {
					frames = int(fv)
				}
			}
		}
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			if frames <= 0 {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.CooldownComponent.Kind(), &component.Cooldown{Frames: frames})
		}
	},
	"cooldown": func(arg any) Action {
		frames := 0
		if arg != nil {
			switch v := arg.(type) {
			case int:
				frames = v
			case float64:
				frames = int(v)
			case map[string]any:
				if fv, ok := v["frames"].(float64); ok {
					frames = int(fv)
				}
			}
		}
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			if frames <= 0 {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.CooldownComponent.Kind(), &component.Cooldown{Frames: frames})
		}
	},
	"tick_timer": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.Context == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.Context.Timer -= 1 / ebiten.ActualTPS()
			if ctx.Context.Timer <= 0 {
				ctx.EnqueueEvent(component.EventID("timer_expired"))
			}
		}
	},
	"emit_event": func(arg any) Action {
		name := fmt.Sprint(arg)
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.EnqueueEvent == nil {
				return
			}
			ctx.EnqueueEvent(component.EventID(name))
		}
	},
	"add_white_flash": func(arg any) Action {
		// arg may be a number (frames) or map; we accept numeric frames and use a default interval
		frames := 30
		if arg != nil {
			switch v := arg.(type) {
			case int:
				frames = v
			case float64:
				frames = int(v)
			}
		}
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.WhiteFlashComponent.Kind(), &component.WhiteFlash{Frames: frames, Interval: 5, Timer: 0, On: true})
		}
	},
	"add_invulnerable": func(arg any) Action {
		// arg may be a number of frames to apply
		frames := 0
		if arg != nil {
			switch v := arg.(type) {
			case int:
				frames = v
			case float64:
				frames = int(v)
			}
		}
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.InvulnerableComponent.Kind(), &component.Invulnerable{Frames: frames})
		}
	},
	"remove_invulnerable": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Remove(ctx.World, ctx.Entity, component.InvulnerableComponent.Kind())
		}
	},
	"destroy_self": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}

			fmt.Println("destroying entity", ctx.Entity)
			if ok := ecs.DestroyEntity(ctx.World, ctx.Entity); !ok {
				panic(fmt.Sprintf("ai: failed to destroy entity %d", ctx.Entity))
			}
		}
	},
	"disable_hazard": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			_ = ecs.Remove(ctx.World, ctx.Entity, component.HazardComponent.Kind())
		}
	},
	"enable_hazard": func(arg any) Action {
		// Enables hazard by adding a Hazard component. Arg may be a map with
		// width/height/offset values or nil to add a default placeholder.
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			var h component.Hazard
			if m, ok := arg.(map[string]any); ok {
				if v, ok2 := m["width"].(float64); ok2 {
					h.Width = v
				}
				if v, ok2 := m["height"].(float64); ok2 {
					h.Height = v
				}
				if v, ok2 := m["offset_x"].(float64); ok2 {
					h.OffsetX = v
				}
				if v, ok2 := m["offset_y"].(float64); ok2 {
					h.OffsetY = v
				}
			}
			_ = ecs.Add(ctx.World, ctx.Entity, component.HazardComponent.Kind(), &h)
		}
	},
	"set_ai": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.AI == nil {
				return
			}
			m, ok := arg.(map[string]any)
			if !ok {
				return
			}
			if v, ok := numberFromMap(m, "move_speed"); ok {
				ctx.AI.MoveSpeed = v
			}
			if v, ok := numberFromMap(m, "follow_range"); ok {
				ctx.AI.FollowRange = v
			}
			if v, ok := numberFromMap(m, "attack_range"); ok {
				ctx.AI.AttackRange = v
			}
			if v, ok := intFromMap(m, "attack_frames"); ok {
				ctx.AI.AttackFrames = v
			}
			if ctx.World != nil {
				_ = ecs.Add(ctx.World, ctx.Entity, component.AIComponent.Kind(), ctx.AI)
			}
		}
	},
	"arena_set_active": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			group, value, ok := parseArenaToggleArg(arg)
			if !ok {
				return
			}
			ecs.ForEach(ctx.World, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
				if node == nil || node.Group != group {
					return
				}
				node.Active = value
				_ = ecs.Add(ctx.World, ent, component.ArenaNodeComponent.Kind(), node)
			})
		}
	},
	"arena_set_hazard": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			group, value, ok := parseArenaToggleArg(arg)
			if !ok {
				return
			}
			ecs.ForEach(ctx.World, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
				if node == nil || node.Group != group {
					return
				}
				node.HazardEnabled = value
				_ = ecs.Add(ctx.World, ent, component.ArenaNodeComponent.Kind(), node)
			})
		}
	},
	"arena_set_transition": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			group, value, ok := parseArenaToggleArg(arg)
			if !ok {
				return
			}
			ecs.ForEach(ctx.World, component.ArenaNodeComponent.Kind(), func(ent ecs.Entity, node *component.ArenaNode) {
				if node == nil || node.Group != group {
					return
				}
				node.TransitionEnabled = value
				_ = ecs.Add(ctx.World, ent, component.ArenaNodeComponent.Kind(), node)
			})
		}
	},
	"set_camera_lock": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}

			camEntity, ok := ecs.First(ctx.World, component.CameraComponent.Kind())
			if !ok {
				return
			}

			camComp, ok := ecs.Get(ctx.World, camEntity, component.CameraComponent.Kind())
			if !ok || camComp == nil {
				return
			}

			enabled := false
			hasEnabled := false
			lockX, lockY := 0.0, 0.0
			hasPoint := false

			switch v := arg.(type) {
			case bool:
				enabled = v
				hasEnabled = true
			case map[string]any:
				if b, ok := v["enabled"].(bool); ok {
					enabled = b
					hasEnabled = true
				} else if b, ok := v["value"].(bool); ok {
					enabled = b
					hasEnabled = true
				}

				x, xOK := numberFromMap(v, "x")
				y, yOK := numberFromMap(v, "y")
				if xOK && yOK {
					lockX = x
					lockY = y
					hasPoint = true
				}
			}

			if !hasEnabled {
				return
			}

			camComp.LockEnabled = enabled
			if enabled {
				if hasPoint {
					camComp.LockCenterX = lockX
					camComp.LockCenterY = lockY
					camComp.LockCapture = false
				} else {
					camComp.LockCapture = true
				}
			} else {
				camComp.LockCapture = false
			}
		}
	},
	"stop_player_input": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			if p, ok := ecs.First(ctx.World, component.PlayerTagComponent.Kind()); ok {
				input, _ := ecs.Get(ctx.World, p, component.InputComponent.Kind())
				if input != nil {
					input.Disabled = true
				}
			}
		}
	},
	"restore_player_input": func(_ any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			if p, ok := ecs.First(ctx.World, component.PlayerTagComponent.Kind()); ok {
				input, _ := ecs.Get(ctx.World, p, component.InputComponent.Kind())
				if input != nil {
					input.Disabled = false
				}
			}
		}
	},
	"camera_shake": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}
			m, ok := arg.(map[string]any)
			if !ok {
				return
			}
			frames, ok := intFromMap(m, "frames")
			if !ok || frames <= 0 {
				frames = 60
			}
			intensity, ok := numberFromMap(m, "intensity")
			if !ok || intensity == 0 {
				intensity = 3
			}
			if camEnt, ok := ecs.First(ctx.World, component.CameraComponent.Kind()); ok {
				_ = ecs.Add(ctx.World, camEnt, component.CameraShakeRequestComponent.Kind(), &component.CameraShakeRequest{Frames: frames, Intensity: intensity})
			}
		}
	},
	"move_forward": func(arg any) Action {
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.GetVelocity == nil || ctx.SetVelocity == nil {
				return
			}

			// determine desired speed: use arg if provided, otherwise fall back to AI.MoveSpeed
			var dx float64
			if arg == nil {
				if ctx.AI != nil {
					dx = ctx.AI.MoveSpeed
				} else {
					dx = 0
				}
			} else {
				dx = asFloat(arg)
			}

			sprite, ok := ecs.Get(ctx.World, ctx.Entity, component.SpriteComponent.Kind())
			if !ok || sprite == nil {
				return
			}

			forward := 1
			if sprite.FacingLeft {
				forward = -1
			}

			_, y := ctx.GetVelocity()
			ctx.SetVelocity(dx*float64(forward), y)
		}
	},
	"instantiate_shockwave": func(arg any) Action {
		// arg may be a map with x,y coordinates and a dir/facing_left value
		marg := arg
		return func(ctx *AIActionContext) {
			if ctx == nil || ctx.World == nil {
				return
			}

			// default to caller position if provided
			x, y := 0.0, 0.0
			gotPos := false
			if m, ok := marg.(map[string]any); ok {
				if vx, ok2 := m["x"].(float64); ok2 {
					x = vx
					gotPos = true
				}
				if vy, ok2 := m["y"].(float64); ok2 {
					y = vy
					gotPos = true
				}
				// accept ints
				if !gotPos {
					if ix, ok2 := m["x"].(int); ok2 {
						x = float64(ix)
						gotPos = true
					}
					if iy, ok2 := m["y"].(int); ok2 {
						y = float64(iy)
						gotPos = true
					}
				}
			}
			if !gotPos && ctx.GetPosition != nil {
				x, y = ctx.GetPosition()
			}

			// determine facing
			facingLeft := false
			if m, ok := marg.(map[string]any); ok {
				if b, ok2 := m["facing_left"].(bool); ok2 {
					facingLeft = b
				} else if s, ok2 := m["dir"].(string); ok2 {
					if strings.ToLower(s) == "left" {
						facingLeft = true
					}
				} else if n, ok2 := m["dir"].(float64); ok2 {
					if n < 0 {
						facingLeft = true
					}
				}
			}

			// build shockwave prefab
			ent, err := entitypkg.BuildEntity(ctx.World, "shockwave.yaml")
			if err != nil {
				panic("ai: instantiate_shockwave: " + err.Error())
			}

			// set transform
			tf, ok := ecs.Get(ctx.World, ent, component.TransformComponent.Kind())
			if !ok || tf == nil {
				tf = &component.Transform{ScaleX: 1, ScaleY: 1}
			}
			tf.X = x
			tf.Y = y

			// set sprite facing if present
			if sp, ok := ecs.Get(ctx.World, ent, component.SpriteComponent.Kind()); ok && sp != nil {
				sp.FacingLeft = facingLeft
			}
		}
	},
}

type TransitionChecker func(ctx *AIActionContext) bool

type TransitionCheckerDef struct {
	From  component.StateID
	Event component.EventID
	Check TransitionChecker
}

var transitionRegistry = map[string]func(any) TransitionChecker{
	"always": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool { return true }
	},
	"sees_player": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil {
				return false
			}
			if !ctx.PlayerFound {
				return false
			}
			if ctx.AI.FollowRange <= 0 {
				return false
			}
			// Prefer using the full 2D distance between AI and player.
			ex2, ey2 := ctx.GetPosition()
			dx := ctx.PlayerX - ex2
			dy := ctx.PlayerY - ey2
			if math.Hypot(dx, dy) > ctx.AI.FollowRange {
				return false
			}

			// Line-of-sight: ensure nothing static blocks view between AI and player.
			if ctx.World != nil {
				_, _, hasHit, _ := firstStaticHit(ctx.World, ctx.PlayerEntity, ex2, ey2, ctx.PlayerX, ctx.PlayerY)
				if hasHit {
					return false
				}
			}
			return true
		}
	},
	"sees_player_and_not_reached_edge": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil {
				return false
			}
			if !ctx.PlayerFound {
				return false
			}
			if ctx.AI.FollowRange <= 0 {
				return false
			}

			nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind())
			if !ok || nav == nil {
				// no nav info: fall back to simple sees_player
				return true
			}

			ex, _ := ctx.GetPosition()
			if (!nav.GroundAheadLeft && ctx.PlayerX < ex) || (!nav.GroundAheadRight && ctx.PlayerX > ex) {
				return false
			}

			// Prefer using the full 2D distance between AI and player.
			// getPosition returns the AI's position; use PlayerY from context.
			ex2, ey2 := ctx.GetPosition()
			dx := ctx.PlayerX - ex2
			dy := ctx.PlayerY - ey2
			if math.Hypot(dx, dy) > ctx.AI.FollowRange {
				return false
			}

			// Line-of-sight: ensure nothing static blocks view between AI and player.
			if ctx.World != nil {
				_, _, hasHit, _ := firstStaticHit(ctx.World, ctx.PlayerEntity, ex2, ey2, ctx.PlayerX, ctx.PlayerY)
				if hasHit {
					return false
				}
			}

			return true
		}
	},
	"loses_player": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil {
				return false
			}
			// if player not found, it's a loss
			if !ctx.PlayerFound {
				return true
			}
			if ctx.AI.FollowRange <= 0 {
				return false
			}
			ex, _ := ctx.GetPosition()
			dx, dy := ctx.PlayerX-ex, ctx.PlayerY-0
			ex2, ey2 := ctx.GetPosition()
			dx = ctx.PlayerX - ex2
			dy = ctx.PlayerY - ey2
			return math.Hypot(dx, dy) > ctx.AI.FollowRange
		}
	},
	"in_attack_range": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil || !ctx.PlayerFound {
				return false
			}
			ex, ey := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dy := ctx.PlayerY - ey
			attackEnterRange := ctx.AI.AttackRange + 24
			if attackEnterRange < 24 {
				attackEnterRange = 24
			}
			return math.Hypot(dx, dy) <= attackEnterRange
		}
	},
	"out_attack_range": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil || ctx.GetPosition == nil || !ctx.PlayerFound {
				return false
			}
			ex, ey := ctx.GetPosition()
			dx := ctx.PlayerX - ex
			dy := ctx.PlayerY - ey
			attackEnterRange := ctx.AI.AttackRange + 24
			attackExitRange := attackEnterRange + 10
			return math.Hypot(dx, dy) > attackExitRange
		}
	},
	"timer_expired": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.Context == nil {
				return false
			}
			res := ctx.Context.Timer <= 0
			return res
		}
	},
	"cooldown_finished": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.World == nil {
				return false
			}
			if _, ok := ecs.Get(ctx.World, ctx.Entity, component.CooldownComponent.Kind()); !ok {
				return true
			}
			return false
		}
	},
	"out_of_health": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.AI == nil {
				return false
			}
			health, _ := ecs.Get(ctx.World, ctx.Entity, component.HealthComponent.Kind())
			return health.Current <= 0
		}
	},
	"reached_edge": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			nav, ok := ecs.Get(ctx.World, ctx.Entity, component.AINavigationComponent.Kind())
			if !ok || nav == nil {
				return false
			}

			return !nav.GroundAheadLeft || !nav.GroundAheadRight
		}
	},
	"animation_finished": func(arg any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.World == nil {
				return false
			}
			anim, ok := ecs.Get(ctx.World, ctx.Entity, component.AnimationComponent.Kind())
			if !ok || anim == nil {
				return false
			}
			return !anim.Playing && anim.Frame == anim.Defs[anim.Current].FrameCount-1
		}
	},
	"is_grounded": func(a any) TransitionChecker {
		return func(ctx *AIActionContext) bool {
			if ctx == nil || ctx.World == nil {
				return false
			}
			// Only applicable to AI entities.
			if !ecs.Has(ctx.World, ctx.Entity, component.AITagComponent.Kind()) {
				return false
			}
			// If AINavigation exists, we can conservatively consider the AI "grounded"
			// when there's ground ahead on either side and a short raycast confirms
			// solid ground beneath the entity. Prefer raycast + physics body.
			body, ok := ecs.Get(ctx.World, ctx.Entity, component.PhysicsBodyComponent.Kind())
			if !ok || body == nil {
				return false
			}
			if ctx.GetPosition == nil {
				return false
			}
			ex, ey := ctx.GetPosition()
			probeDist := 8.0
			if body.Height > 0 {
				probeDist = body.Height/2 + 2
			}
			_, _, hit, _ := firstStaticHit(ctx.World, ctx.Entity, ex, ey, ex, ey+probeDist)
			return hit
		}
	},
}

func asFloat(v any) float64 {
	switch t := v.(type) {
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case float64:
		return t
	case float32:
		return float64(t)
	default:
		return 0
	}
}

func numberFromMap(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func intFromMap(m map[string]any, key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	default:
		return 0, false
	}
}

func parseArenaToggleArg(arg any) (string, bool, bool) {
	m, ok := arg.(map[string]any)
	if !ok {
		return "", false, false
	}
	group, _ := m["group"].(string)
	if group == "" {
		return "", false, false
	}
	if v, ok := m["value"].(bool); ok {
		return group, v, true
	}
	if v, ok := m["active"].(bool); ok {
		return group, v, true
	}
	if v, ok := m["enabled"].(bool); ok {
		return group, v, true
	}
	return "", false, false
}

func CompileFSM(raw prefabs.RawFSM) (*FSMDef, error) {
	if raw.Initial == "" {
		return nil, fmt.Errorf("fsm: missing initial state")
	}

	states := map[component.StateID]StateDef{}
	build := func(list []map[string]any) ([]Action, error) {
		if len(list) == 0 {
			return nil, nil
		}
		out := make([]Action, 0, len(list))
		for _, e := range list {
			for k, v := range e {
				makeAction, ok := actionRegistry[k]
				if !ok {
					return nil, fmt.Errorf("fsm: unknown action %q", k)
				}
				out = append(out, makeAction(v))
			}
		}
		return out, nil
	}

	for name, s := range raw.States {
		onEnter, err := build(s.OnEnter)
		if err != nil {
			return nil, err
		}
		while, err := build(s.While)
		if err != nil {
			return nil, err
		}
		onExit, err := build(s.OnExit)
		if err != nil {
			return nil, err
		}
		states[component.StateID(name)] = StateDef{
			OnEnter: onEnter,
			While:   while,
			OnExit:  onExit,
		}
	}

	transitions := map[component.StateID]map[component.EventID]component.StateID{}
	var checkers []TransitionCheckerDef

	for from, rawVal := range raw.Transitions {
		fromID := component.StateID(from)
		transitions[fromID] = map[component.EventID]component.StateID{}

		switch v := rawVal.(type) {
		case map[string]any:
			for evName, toVal := range v {
				if isConditionExpression(evName) {
					var toState string
					var arg any
					if m, ok := toVal.(map[string]any); ok {
						if ts, ok2 := m["to"].(string); ok2 {
							toState = ts
						}
						arg = m["arg"]
					} else if s, ok2 := toVal.(string); ok2 {
						toState = s
					}
					if toState == "" {
						return nil, fmt.Errorf("fsm: missing to state for transition expression %s.%s", from, evName)
					}
					checker, err := compileTransitionExpression(evName, arg)
					if err != nil {
						return nil, fmt.Errorf("fsm: compile transition expression %s.%s: %w", from, evName, err)
					}
					eid := component.EventID(fmt.Sprintf("__cond_%s_expr_%s", from, evName))
					transitions[fromID][eid] = component.StateID(toState)
					checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: checker})
					continue
				}

				// simple mapping: event -> state
				if toStr, ok := toVal.(string); ok {
					transitions[fromID][component.EventID(evName)] = component.StateID(toStr)
					continue
				}
				// registry-driven transition: evName is a condition name
				if maker, ok := transitionRegistry[evName]; ok {
					var toState string
					var arg any
					if m, ok := toVal.(map[string]any); ok {
						if ts, ok2 := m["to"].(string); ok2 {
							toState = ts
						}
						arg = m["arg"]
					} else if s, ok2 := toVal.(string); ok2 {
						toState = s
					}
					if toState == "" {
						return nil, fmt.Errorf("fsm: missing to state for transition %s.%s", from, evName)
					}
					eid := component.EventID(fmt.Sprintf("__cond_%s_%s", from, evName))
					transitions[fromID][eid] = component.StateID(toState)
					checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: maker(arg)})
					continue
				}
				return nil, fmt.Errorf("fsm: invalid transition value for %s.%s", from, evName)
			}
		case []any:
			for i, item := range v {
				m, ok := item.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("fsm: invalid transition entry %v", item)
				}
				for key, val := range m {
					if isConditionExpression(key) {
						var toState string
						var arg any
						if mv, ok2 := val.(map[string]any); ok2 {
							if ts, ok3 := mv["to"].(string); ok3 {
								toState = ts
							}
							arg = mv["arg"]
						} else if s, ok3 := val.(string); ok3 {
							toState = s
						}
						if toState == "" {
							return nil, fmt.Errorf("fsm: missing to state for transition expression %s", key)
						}
						checker, err := compileTransitionExpression(key, arg)
						if err != nil {
							return nil, fmt.Errorf("fsm: compile transition expression %s: %w", key, err)
						}
						eid := component.EventID(fmt.Sprintf("__cond_%s_%d", from, i))
						transitions[fromID][eid] = component.StateID(toState)
						checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: checker})
						continue
					}

					if maker, ok := transitionRegistry[key]; ok {
						var toState string
						var arg any
						if mv, ok2 := val.(map[string]any); ok2 {
							if ts, ok3 := mv["to"].(string); ok3 {
								toState = ts
							}
							arg = mv["arg"]
						} else if s, ok3 := val.(string); ok3 {
							toState = s
						}
						if toState == "" {
							return nil, fmt.Errorf("fsm: missing to state for transition %s", key)
						}
						eid := component.EventID(fmt.Sprintf("__cond_%s_%d", from, i))
						transitions[fromID][eid] = component.StateID(toState)
						checkers = append(checkers, TransitionCheckerDef{From: fromID, Event: eid, Check: maker(arg)})
					} else {
						if toState, ok2 := val.(string); ok2 {
							transitions[fromID][component.EventID(key)] = component.StateID(toState)
						} else {
							return nil, fmt.Errorf("fsm: invalid transition mapping for %s -> %v", key, val)
						}
					}
				}
			}
		default:
			return nil, fmt.Errorf("fsm: invalid transitions type for state %s", from)
		}
	}

	return &FSMDef{
		Initial:     component.StateID(raw.Initial),
		States:      states,
		Transitions: transitions,
		Checkers:    checkers,
	}, nil
}

func LoadFSMFromPrefab(path string) (*FSMDef, error) {
	raw, err := prefabs.LoadRawFSM(path)
	if err != nil {
		return nil, err
	}
	return CompileFSM(raw)
}

func DefaultEnemyFSM() *FSMDef {
	f := &FSMDef{
		Initial: component.StateID("idle"),
		States: map[component.StateID]StateDef{
			component.StateID("idle"): {
				OnEnter: []Action{actionRegistry["set_animation"]("idle")},
				While:   []Action{actionRegistry["stop_x"](nil)},
			},
			component.StateID("follow"): {
				OnEnter: []Action{actionRegistry["set_animation"]("run")},
				While: []Action{
					actionRegistry["move_towards_player"](nil),
					actionRegistry["face_player"](nil),
				},
			},
			component.StateID("attack"): {
				OnEnter: []Action{
					actionRegistry["set_animation"]("attack"),
					actionRegistry["start_attack_timer"](nil),
				},
				While: []Action{
					actionRegistry["stop_x"](nil),
					actionRegistry["tick_timer"](nil),
				},
			},
		},
		Transitions: map[component.StateID]map[component.EventID]component.StateID{
			component.StateID("idle"): {
				component.EventID("sees_player"): component.StateID("follow"),
			},
			component.StateID("follow"): {
				component.EventID("loses_player"): component.StateID("idle"),
			},
			component.StateID("attack"): {
				component.EventID("timer_expired"): component.StateID("follow"),
				component.EventID("loses_player"):  component.StateID("idle"),
			},
		},
	}

	return f
}

func CompileFSMSpec(spec component.AIFSMSpec) (*FSMDef, error) {
	raw := prefabs.RawFSM{
		Initial:     spec.Initial,
		States:      map[string]prefabs.RawState{},
		Transitions: map[string]any{},
	}
	// copy transitions into the flexible raw.Transitions shape
	// spec.Transitions is an ordered slice of maps (to preserve priority),
	// so convert each entry into a []any of map[string]any to feed CompileFSM.
	for from, evs := range spec.Transitions {
		items := make([]any, 0, len(evs))
		for _, evmap := range evs {
			m := map[string]any{}
			for k, v := range evmap {
				m[k] = v
			}
			items = append(items, m)
		}
		raw.Transitions[from] = items
	}
	for name, s := range spec.States {
		raw.States[name] = prefabs.RawState{
			OnEnter: s.OnEnter,
			While:   s.While,
			OnExit:  s.OnExit,
		}
	}
	return CompileFSM(raw)
}

type exprTokenType int

const (
	exprTokenIdentifier exprTokenType = iota
	exprTokenNot
	exprTokenAnd
	exprTokenOr
	exprTokenLParen
	exprTokenRParen
)

type exprToken struct {
	typ exprTokenType
	val string
}

type transitionExprParser struct {
	tokens []exprToken
	pos    int
	arg    any
}

func isConditionExpression(s string) bool {
	return strings.Contains(s, "&&") || strings.Contains(s, "||") || strings.Contains(s, "(") || strings.Contains(s, ")") || strings.Contains(s, "!")
}

func compileTransitionExpression(expr string, arg any) (TransitionChecker, error) {
	tokens, err := tokenizeTransitionExpression(expr)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty expression")
	}

	p := &transitionExprParser{tokens: tokens, arg: arg}
	checker, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.tokens) {
		return nil, fmt.Errorf("unexpected token %q", p.tokens[p.pos].val)
	}
	return checker, nil
}

func tokenizeTransitionExpression(expr string) ([]exprToken, error) {
	tokens := make([]exprToken, 0, 8)
	for i := 0; i < len(expr); {
		r := rune(expr[i])
		if unicode.IsSpace(r) {
			i++
			continue
		}
		if i+1 < len(expr) && expr[i] == '&' && expr[i+1] == '&' {
			tokens = append(tokens, exprToken{typ: exprTokenAnd, val: "&&"})
			i += 2
			continue
		}
		if i+1 < len(expr) && expr[i] == '|' && expr[i+1] == '|' {
			tokens = append(tokens, exprToken{typ: exprTokenOr, val: "||"})
			i += 2
			continue
		}
		if expr[i] == '(' {
			tokens = append(tokens, exprToken{typ: exprTokenLParen, val: "("})
			i++
			continue
		}
		if expr[i] == ')' {
			tokens = append(tokens, exprToken{typ: exprTokenRParen, val: ")"})
			i++
			continue
		}
		if expr[i] == '!' {
			tokens = append(tokens, exprToken{typ: exprTokenNot, val: "!"})
			i++
			continue
		}

		start := i
		for i < len(expr) {
			if i+1 < len(expr) && ((expr[i] == '&' && expr[i+1] == '&') || (expr[i] == '|' && expr[i+1] == '|')) {
				break
			}
			if expr[i] == '(' || expr[i] == ')' || expr[i] == '!' {
				break
			}
			i++
		}
		ident := strings.TrimSpace(expr[start:i])
		if ident == "" {
			return nil, fmt.Errorf("invalid token near %q", expr[start:])
		}
		tokens = append(tokens, exprToken{typ: exprTokenIdentifier, val: ident})
	}
	return tokens, nil
}

func (p *transitionExprParser) parseOr() (TransitionChecker, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.pos < len(p.tokens) && p.tokens[p.pos].typ == exprTokenOr {
		p.pos++
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		l := left
		r := right
		left = func(ctx *AIActionContext) bool {
			return l(ctx) || r(ctx)
		}
	}
	return left, nil
}

func (p *transitionExprParser) parseAnd() (TransitionChecker, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for p.pos < len(p.tokens) && p.tokens[p.pos].typ == exprTokenAnd {
		p.pos++
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		l := left
		r := right
		left = func(ctx *AIActionContext) bool {
			return l(ctx) && r(ctx)
		}
	}
	return left, nil
}

func (p *transitionExprParser) parsePrimary() (TransitionChecker, error) {
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	tok := p.tokens[p.pos]
	switch tok.typ {
	case exprTokenIdentifier:
		p.pos++
		maker, ok := transitionRegistry[tok.val]
		if !ok {
			return nil, fmt.Errorf("unknown transition condition %q", tok.val)
		}
		return maker(argForCondition(tok.val, p.arg)), nil
	case exprTokenNot:
		p.pos++
		inner, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return func(ctx *AIActionContext) bool {
			return !inner(ctx)
		}, nil
	case exprTokenLParen:
		p.pos++
		inner, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.pos >= len(p.tokens) || p.tokens[p.pos].typ != exprTokenRParen {
			return nil, fmt.Errorf("missing closing parenthesis")
		}
		p.pos++
		return inner, nil
	default:
		return nil, fmt.Errorf("unexpected token %q", tok.val)
	}
}

func argForCondition(name string, arg any) any {
	if m, ok := arg.(map[string]any); ok {
		if argsAny, ok := m["args"]; ok {
			if args, ok2 := argsAny.(map[string]any); ok2 {
				if v, ok3 := args[name]; ok3 {
					return v
				}
			}
		}
		if v, ok := m[name]; ok {
			return v
		}
	}
	return nil
}
