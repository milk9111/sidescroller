package system

import (
	"strings"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const tileSize = 32.0

// TransitionSystem detects when the player enters a transition volume and
// requests a level change by spawning a one-shot request entity.
type TransitionSystem struct{}

func NewTransitionSystem() *TransitionSystem { return &TransitionSystem{} }

func (ts *TransitionSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	// Don't enqueue multiple requests.
	if _, ok := w.First(component.LevelChangeRequestComponent.Kind()); ok {
		return
	}

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
		// Still inside: can't trigger anything yet.
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

		// Basic sanity for authored direction.
		dir := component.TransitionDirection(strings.ToLower(string(tr.EnterDir)))
		switch dir {
		case component.TransitionDirUp, component.TransitionDirDown, component.TransitionDirLeft, component.TransitionDirRight:
			// ok
		default:
			dir = ""
		}

		reqEnt := w.CreateEntity()
		_ = ecs.Add(w, reqEnt, component.LevelChangeRequestComponent, component.LevelChangeRequest{
			TargetLevel:       tr.TargetLevel,
			SpawnTransitionID: tr.LinkedID,
			EnterDir:          dir,
			FromTransitionID:  tr.ID,
			FromTransitionEnt: uint64(ent),
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
