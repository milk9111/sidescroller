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

	// If there's an active runtime component, update its timers/alpha and
	// progress phases. Otherwise, detect player entering a transition and
	// create a runtime to begin the fade-out.
	if rtEnt, ok := w.First(component.TransitionRuntimeComponent.Kind()); ok {
		rt, _ := ecs.Get(w, rtEnt, component.TransitionRuntimeComponent)
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
				// LevelLoadedComponent).
				reqEnt := w.CreateEntity()
				_ = ecs.Add(w, reqEnt, component.LevelChangeRequestComponent, rt.Req)
				rt.ReqSent = true
			}

			// If the request has been sent and the outer Game loop has signalled
			// the level has finished loading (by adding LevelLoadedComponent),
			// move into FadeIn so we can animate back from black.
			if rt.ReqSent {
				if _, loaded := w.First(component.LevelLoadedComponent.Kind()); loaded {
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
				w.DestroyEntity(rtEnt)
				if lvlEnt, ok := w.First(component.LevelLoadedComponent.Kind()); ok {
					w.DestroyEntity(lvlEnt)
				}
			}
		default:
			// shouldn't happen; ensure clean state
			w.DestroyEntity(rtEnt)
		}
		_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent, rt)
		return
	}

	// No active runtime: detect player entering a transition to begin.
	player, ok := w.First(component.PlayerTagComponent.Kind())
	if !ok {
		return
	}
	playerAABB, ok := playerAABB(w, player)
	if !ok {
		return
	}

	// Handle the "spawned inside a transition" lockout.
	cooldown, _ := ecs.Get(w, player, component.TransitionCooldownComponent)
	if cooldown.Active && cooldown.TransitionID != "" {
		inside := false
		for _, ent := range w.Query(component.TransitionComponent.Kind(), component.TransformComponent.Kind()) {
			tr, _ := ecs.Get(w, ent, component.TransitionComponent)
			if tr.ID != cooldown.TransitionID {
				continue
			}
			if aabbIntersects(playerAABB, transitionAABB(w, ent, tr)) {
				inside = true
				break
			}
		}
		if !inside {
			cooldown.Active = false
			cooldown.TransitionID = ""
			_ = ecs.Add(w, player, component.TransitionCooldownComponent, cooldown)
		}
		if inside {
			return
		}
	}

	for _, ent := range w.Query(component.TransitionComponent.Kind(), component.TransformComponent.Kind()) {
		tr, ok := ecs.Get(w, ent, component.TransitionComponent)
		if !ok {
			continue
		}
		if tr.TargetLevel == "" {
			continue
		}
		if tr.LinkedID == "" {
			continue
		}
		if !aabbIntersects(playerAABB, transitionAABB(w, ent, tr)) {
			continue
		}

		dir := component.TransitionDirection(strings.ToLower(string(tr.EnterDir)))
		switch dir {
		case component.TransitionDirUp, component.TransitionDirDown, component.TransitionDirLeft, component.TransitionDirRight:
		default:
			dir = ""
		}

		// Create a transient runtime entity that will manage fade-out/in.
		rtEnt := w.CreateEntity()
		_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent, component.TransitionRuntime{
			Phase: component.TransitionFadeOut,
			Alpha: 0,
			Timer: transitionFadeFrames,
			Req: component.LevelChangeRequest{
				TargetLevel:       tr.TargetLevel,
				SpawnTransitionID: tr.LinkedID,
				EnterDir:          dir,
				FromTransitionID:  tr.ID,
				FromTransitionEnt: uint64(ent),
			},
			ReqSent: false,
		})
		return
	}
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

func transitionAABB(w *ecs.World, ent ecs.Entity, tr component.Transition) aabb {
	transform, _ := ecs.Get(w, ent, component.TransformComponent)
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
	transform, ok := ecs.Get(w, player, component.TransformComponent)
	if !ok {
		return aabb{}, false
	}
	body, ok := ecs.Get(w, player, component.PhysicsBodyComponent)
	if !ok {
		return aabb{}, false
	}
	if body.Width <= 0 || body.Height <= 0 {
		return aabb{}, false
	}
	return aabb{x: transform.X, y: transform.Y, w: body.Width, h: body.Height}, true
}
