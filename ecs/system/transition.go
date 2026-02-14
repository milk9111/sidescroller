package system

import (
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const tileSize = 32.0
const transitionFadeFrames = 30

// TransitionSystem detects when the player enters a transition volume and
// requests a level change by spawning a one-shot request entity.
type TransitionSystem struct{}

func NewTransitionSystem() *TransitionSystem { return &TransitionSystem{} }

func (ts *TransitionSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	// If there's an active runtime Component.Kind(), update its timers/alpha and
	// progress phases. Otherwise, detect player entering a transition and
	// create a runtime to begin the fade-out.
	if rtEnt, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind()); ok {
		rt, _ := ecs.Get(w, rtEnt, component.TransitionRuntimeComponent.Kind())
		if rt.Timer > 0 {
			rt.Timer--
		}
		switch rt.Phase {
		case component.TransitionFadeOut:
			rt.Alpha = 1 - float64(rt.Timer)/float64(transitionFadeFrames)
			if rt.Timer <= 0 && !rt.ReqSent {
				// Send the LevelChangeRequest into the world so the Game loop
				// can perform the IO reload. Keep the runtime alive while we
				// wait for the outer loop to finish loading (signalled by
				// LevelLoadedComponent.Kind()).
				reqEnt := ecs.CreateEntity(w)
				_ = ecs.Add(w, reqEnt, component.LevelChangeRequestComponent.Kind(), &rt.Req)
				rt.ReqSent = true
			}

			// If the request has been sent and the outer Game loop has signalled
			// the level has finished loading (by adding LevelLoadedComponent.Kind()),
			// move into FadeIn so we can animate back from black.
			if rt.ReqSent {
				if _, loaded := ecs.First(w, component.LevelLoadedComponent.Kind()); loaded {
					rt.Phase = component.TransitionFadeIn
					rt.Timer = transitionFadeFrames
					rt.Alpha = 1
				}
			}
		case component.TransitionFadeIn:
			rt.Alpha = float64(rt.Timer) / float64(transitionFadeFrames)
			if rt.Timer <= 0 {
				// Transition complete: remove the runtime and any LevelLoaded
				// markers.
				ecs.DestroyEntity(w, rtEnt)
				if lvlEnt, ok := ecs.First(w, component.LevelLoadedComponent.Kind()); ok {
					ecs.DestroyEntity(w, lvlEnt)
				}
			}
		default:
			// shouldn't happen; ensure clean state
			ecs.DestroyEntity(w, rtEnt)
		}
		_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent.Kind(), rt)
		return
	}

	// No active runtime: detect player entering a transition to begin.
	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	playerAABB, ok := playerAABB(w, player)
	if !ok {
		return
	}

	// Handle the "spawned inside a transition" lockout.
	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	if !ok || cooldown == nil {
		cooldown = &component.TransitionCooldown{}
		_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), cooldown)
	}
	if cooldown.Active && cooldown.TransitionID != "" {
		inside := false
		ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, tr *component.Transition, _ *component.Transform) {
			if tr.ID != cooldown.TransitionID || inside {
				return
			}
			if aabbIntersects(playerAABB, transitionAABB(w, e, tr)) {
				inside = true
			}
		})

		if !inside {
			cooldown.Active = false
			cooldown.TransitionID = ""
			// _ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), cooldown)
		}

		if inside {
			return
		}
	}

	createdTransition := false
	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(ent ecs.Entity, tr *component.Transition, _ *component.Transform) {
		if createdTransition || tr.TargetLevel == "" || tr.LinkedID == "" || !aabbIntersects(playerAABB, transitionAABB(w, ent, tr)) {
			return
		}

		dir := component.TransitionDirection(strings.ToLower(string(tr.EnterDir)))

		// Create a transient runtime entity that will manage fade-out/in.
		rtEnt := ecs.CreateEntity(w)
		// Capture player's facing at the time of entering the transition so the
		// spawn side can reuse it after the level reload.
		facingLeft := false
		if spriteComp, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && spriteComp != nil {
			facingLeft = spriteComp.FacingLeft
		}
		// Determine whether the player was below the transition area when
		// triggering; if so, record that so the spawn handler can apply a
		// pop upward after the new level is loaded.
		trAABB := transitionAABB(w, ent, tr)
		playerCenterY := playerAABB.y + playerAABB.h/2.0
		transitionCenterY := trAABB.y + trAABB.h/2.0
		entryFromBelow := playerCenterY > transitionCenterY

		_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent.Kind(), &component.TransitionRuntime{
			Phase: component.TransitionFadeOut,
			Alpha: 0,
			Timer: transitionFadeFrames,
			Req: component.LevelChangeRequest{
				TargetLevel:       tr.TargetLevel,
				SpawnTransitionID: tr.LinkedID,
				EnterDir:          dir,
				FromFacingLeft:    facingLeft,
				FromTransitionID:  tr.ID,
				FromTransitionEnt: uint64(ent),
				EntryFromBelow:    entryFromBelow,
			},
			ReqSent: false,
		})

		createdTransition = true
	})
}

type aabb struct {
	x float64
	y float64
	w float64
	h float64
}

func aabbIntersects(a, b aabb) bool {
	return a.x < b.x+b.w &&
		a.x+a.w > b.x &&
		a.y < b.y+b.h &&
		a.y+a.h > b.y
}

func transitionAABB(w *ecs.World, ent ecs.Entity, tr *component.Transition) aabb {
	transform, _ := ecs.Get(w, ent, component.TransformComponent.Kind())
	wid := tr.Bounds.W
	hei := tr.Bounds.H
	if wid <= 0 {
		wid = tileSize
	}
	if hei <= 0 {
		hei = tileSize
	}
	if wid < tileSize {
		wid = tileSize
	}
	if hei < tileSize {
		hei = tileSize
	}
	return aabb{
		x: transform.X + tr.Bounds.X,
		y: transform.Y + tr.Bounds.Y,
		w: wid,
		h: hei,
	}
}

func playerAABB(w *ecs.World, player ecs.Entity) (aabb, bool) {
	transform, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok {
		return aabb{}, false
	}
	body, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind())
	if !ok {
		return aabb{}, false
	}
	if body.Width <= 0 || body.Height <= 0 {
		return aabb{}, false
	}
	return aabb{x: transform.X, y: transform.Y, w: body.Width, h: body.Height}, true
}
