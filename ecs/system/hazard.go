package system

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type HazardSystem struct{}

func NewHazardSystem() *HazardSystem { return &HazardSystem{} }

type hazardAABB struct {
	x float64
	y float64
	w float64
	h float64
}

type hazardHitSource struct {
	bounds  hazardAABB
	centerX float64
	centerY float64
	entity  ecs.Entity
}

func overlapsAABB(a, b hazardAABB) bool {
	return a.x < b.x+b.w && a.x+a.w > b.x && a.y < b.y+b.h && a.y+a.h > b.y
}

func physicsBodyAABB(w *ecs.World, e ecs.Entity, t *component.Transform, b *component.PhysicsBody) (hazardAABB, bool) {
	if t == nil || b == nil {
		return hazardAABB{}, false
	}
	width := b.Width
	height := b.Height
	if width <= 0 || height <= 0 {
		return hazardAABB{}, false
	}
	x := aabbTopLeftX(w, e, t.X, b.OffsetX, width, b.AlignTopLeft)
	if b.AlignTopLeft {
		return hazardAABB{x: x, y: t.Y + b.OffsetY, w: width, h: height}, true
	}
	return hazardAABB{x: x, y: t.Y + b.OffsetY - height/2, w: width, h: height}, true
}

func hazardBounds(w *ecs.World, e ecs.Entity, h *component.Hazard, t *component.Transform) (hazardAABB, bool) {
	if h == nil || t == nil || h.Width <= 0 || h.Height <= 0 {
		return hazardAABB{}, false
	}

	// Default: treat transform (t.X,t.Y) as the sprite transform point and
	// interpret hazard offsets relative to that point. Prefer to align the
	// hazard top-left to the sprite's rendered top-left when a Sprite
	// component is present.
	x := t.X + facingAdjustedOffsetX(w, e, h.OffsetX, h.Width, true)
	y := t.Y + h.OffsetY
	wid := h.Width
	hgt := h.Height

	if s, ok := ecs.Get(w, e, component.SpriteComponent.Kind()); ok && s != nil && s.Image != nil {
		// Determine sprite source size
		imgW := s.Image.Bounds().Dx()
		imgH := s.Image.Bounds().Dy()
		if s.UseSource {
			imgW = s.Source.Dx()
			imgH = s.Source.Dy()
		}

		// scaled origin
		originX := s.OriginX * t.ScaleX
		originY := s.OriginY * t.ScaleY

		x = t.X - originX + facingAdjustedOffsetX(w, e, h.OffsetX, wid, true)
		y = t.Y - originY + h.OffsetY

		// If spec provided different hazard size, keep it; otherwise use sprite pixel size
		if wid <= 0 {
			wid = float64(imgW) * t.ScaleX
		}
		if hgt <= 0 {
			hgt = float64(imgH) * t.ScaleY
		}
	}

	// If there's no rotation, return the simple AABB.
	if t.Rotation == 0 {
		return hazardAABB{x: x, y: y, w: wid, h: hgt}, true
	}

	// Rotate the four corners of the hazard rect around the transform origin
	// (t.X, t.Y) and compute the axis-aligned bounding box that contains
	// the rotated rectangle. This ensures the collider covers the rotated
	// sprite area for hazard checks.
	cx := t.X
	cy := t.Y
	cosR := math.Cos(t.Rotation)
	sinR := math.Sin(t.Rotation)

	corners := [4][2]float64{
		{x, y},
		{x + wid, y},
		{x, y + hgt},
		{x + wid, y + hgt},
	}

	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)
	for _, c := range corners {
		dx := c[0] - cx
		dy := c[1] - cy
		rx := dx*cosR - dy*sinR + cx
		ry := dx*sinR + dy*cosR + cy
		if rx < minX {
			minX = rx
		}
		if ry < minY {
			minY = ry
		}
		if rx > maxX {
			maxX = rx
		}
		if ry > maxY {
			maxY = ry
		}
	}

	return hazardAABB{x: minX, y: minY, w: maxX - minX, h: maxY - minY}, true
}

// DrawHazardDebug renders hazard bounds for debug visualization.
func DrawHazardDebug(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}
	camX, camY, zoom := debugCameraTransform(w)
	ecs.ForEach2(w, component.HazardComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, h *component.Hazard, t *component.Transform) {
		if h == nil || t == nil {
			return
		}
		if b, ok := hazardBounds(w, e, h, t); ok {
			x := (b.x - camX) * zoom
			y := (b.y - camY) * zoom
			wdt := b.w * zoom
			hgt := b.h * zoom
			// semi-transparent fill + outline
			vector.FillRect(screen, float32(x), float32(y), float32(wdt), float32(hgt), color.RGBA{R: 255, G: 0, B: 0, A: 48}, false)
			vector.StrokeRect(screen, float32(x), float32(y), float32(wdt), float32(hgt), 1.0, color.RGBA{R: 255, G: 0, B: 0, A: 200}, false)
		}
	})
}

func (s *HazardSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	hazards := make([]hazardHitSource, 0, 16)
	seenHazards := make(map[ecs.Entity]struct{}, 16)
	ecs.ForEach2(w, component.HazardComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, h *component.Hazard, t *component.Transform) {
		if h == nil || t == nil {
			return
		}
		if _, exists := seenHazards[e]; exists {
			return
		}
		seenHazards[e] = struct{}{}

		if b, ok := hazardBounds(w, e, h, t); ok {
			hazards = append(hazards, hazardHitSource{bounds: b, centerX: b.x + b.w/2, centerY: b.y + b.h/2, entity: e})
		}
	})

	if player, ok := ecs.First(w, component.PlayerTagComponent.Kind()); ok {
		t, tok := ecs.Get(w, player, component.TransformComponent.Kind())
		body, bok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind())
		playerOverHazard := false

		if tok && bok && t != nil && body != nil {
			if playerBox, ok := physicsBodyAABB(w, player, t, body); ok {
				for _, hz := range hazards {
					if hz.entity == player {
						// ignore hazards originating from the player itself
						continue
					}
					if overlapsAABB(playerBox, hz.bounds) {
						playerOverHazard = true
						// If player is currently invulnerable, ignore hazard hits.
						if ecs.Has(w, player, component.InvulnerableComponent.Kind()) {
							continue
						}
						// Immediately mark player invulnerable to avoid multiple
						// damage applications within the same frame. Give a
						// single-frame timed invulnerability so the invuln system
						// can remove it next tick automatically.
						_ = ecs.Add(w, player, component.InvulnerableComponent.Kind(), &component.Invulnerable{Frames: 1})
						s.applyPlayerHazardHit(w, player, hz.centerX, hz.centerY, hz.entity)
						break
					}
				}
			}
		}

		// Track last safe grounded position for player. Do not update while the
		// player is currently overlapping a hazard (e.g. standing on spikes).
		if tok && t != nil {
			safe, hasSafe := ecs.Get(w, player, component.SafeRespawnComponent.Kind())
			if !hasSafe || safe == nil {
				safe = &component.SafeRespawn{X: t.X, Y: t.Y, Initialized: true}
			} else if !safe.Initialized {
				safe.X = t.X
				safe.Y = t.Y
				safe.Initialized = true
			}
			if pc, ok := ecs.Get(w, player, component.PlayerCollisionComponent.Kind()); ok && pc != nil {
				if (pc.Grounded || pc.GroundGrace > 0) && !playerOverHazard {
					safe.X = t.X
					safe.Y = t.Y
					safe.Initialized = true
				}
			}
			_ = ecs.Add(w, player, component.SafeRespawnComponent.Kind(), safe)
		}
	}

	if len(hazards) == 0 {
		return
	}

	enemyHit := make(map[ecs.Entity]struct{}, 8)
	ecs.ForEach3(w, component.AITagComponent.Kind(), component.TransformComponent.Kind(), component.PhysicsBodyComponent.Kind(), func(e ecs.Entity, _ *component.AITag, t *component.Transform, body *component.PhysicsBody) {
		if t == nil || body == nil {
			return
		}
		if _, seen := enemyHit[e]; seen {
			return
		}
		box, ok := physicsBodyAABB(w, e, t, body)
		if !ok {
			return
		}
		for _, hz := range hazards {
			if hz.entity == e {
				// don't let an enemy's own hazard kill itself
				continue
			}
			// If both the hazard source and the target are AI, skip applying
			// damage so enemies do not kill each other.
			if hz.entity != 0 && ecs.Has(w, hz.entity, component.AITagComponent.Kind()) && ecs.Has(w, e, component.AITagComponent.Kind()) {
				continue
			}
			if overlapsAABB(box, hz.bounds) {
				enemyHit[e] = struct{}{}
				s.killEnemyOnHazard(w, e, hz.centerX, hz.centerY)
				break
			}
		}
	})
}

func (s *HazardSystem) applyPlayerHazardHit(w *ecs.World, player ecs.Entity, sourceX, sourceY float64, sourceEntity ecs.Entity) {
	health, hok := ecs.Get(w, player, component.HealthComponent.Kind())
	if hok && health != nil {
		health.Current--
		if health.Current < 0 {
			health.Current = 0
		}
		_ = ecs.Add(w, player, component.HealthComponent.Kind(), health)
		state := "hit"
		if health.Current == 0 {
			state = "death"
		}
		_ = ecs.Add(w, player, component.PlayerStateInterruptComponent.Kind(), &component.PlayerStateInterrupt{State: state})
		// Emit damage knockback request instead of applying immediately.
		req := &component.DamageKnockback{SourceX: sourceX, SourceY: sourceY}
		// If hazard was from an AI, consider strong knockback behavior.
		if sourceEntity != 0 && ecs.Has(w, sourceEntity, component.AITagComponent.Kind()) {
			req.SourceEntity = uint64(sourceEntity)
		}
		_ = ecs.Add(w, player, component.DamageKnockbackRequestComponent.Kind(), req)
	}

	t, tok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !tok || t == nil {
		return
	}
	safe, sok := ecs.Get(w, player, component.SafeRespawnComponent.Kind())
	respawnRequested := false
	shouldRespawn := true
	if sourceEntity != 0 && ecs.Has(w, sourceEntity, component.AITagComponent.Kind()) {
		shouldRespawn = false
	}
	if shouldRespawn && sok && safe != nil && safe.Initialized {
		// If player is anchored, immediately remove anchor constraints from
		// the physics space so the teleport doesn't get resisted by joints.
		// Request that anchors be removed by the PhysicsSystem before
		// respawning the player. PhysicsSystem will process
		// `AnchorPendingDestroyComponent` at the start of its Update.
		ecs.ForEach(w, component.AnchorTagComponent.Kind(), func(e ecs.Entity, _ *component.AnchorTag) {
			_ = ecs.Add(w, e, component.AnchorPendingDestroyComponent.Kind(), &component.AnchorPendingDestroy{})
		})

		// Add a respawn request for the player; a dedicated RespawnSystem
		// (running after PhysicsSystem) will perform the actual teleport
		// after constraints have been removed.
		_ = ecs.Add(w, player, component.RespawnRequestComponent.Kind(), &component.RespawnRequest{})
		respawnRequested = true
	}
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), t)

	if body, bok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); bok && body != nil && body.Body != nil {
		centerX := bodyCenterX(w, player, t, body)
		centerY := t.Y + body.OffsetY
		if body.AlignTopLeft {
			centerY += body.Height / 2
		}
		// Only reset position/velocity when a respawn is actually requested.
		if respawnRequested {
			body.Body.SetPosition(cp.Vector{X: centerX, Y: centerY})
			body.Body.SetVelocityVector(cp.Vector{})
			body.Body.SetAngularVelocity(0)
		}
	}
}

func (s *HazardSystem) killEnemyOnHazard(w *ecs.World, enemy ecs.Entity, sourceX, sourceY float64) {
	if health, ok := ecs.Get(w, enemy, component.HealthComponent.Kind()); ok && health != nil {
		if health.Current > 0 {
			req := &component.DamageKnockback{SourceX: sourceX, SourceY: sourceY}
			_ = ecs.Add(w, enemy, component.DamageKnockbackRequestComponent.Kind(), req)
		}
		health.Current = 0
		_ = ecs.Add(w, enemy, component.HealthComponent.Kind(), health)
	}
	_ = ecs.Add(w, enemy, component.AIStateInterruptComponent.Kind(), &component.AIStateInterrupt{Event: "hit"})
}
