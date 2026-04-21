package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestHealthDeathFadeSystemStartsFadeWithoutDeathAnimation(t *testing.T) {
	w := ecs.NewWorld()
	deathSystem := NewHealthDeathFadeSystem()
	fadeSystem := NewSpriteFadeOutSystem()

	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 0}); err != nil {
		t.Fatalf("add health: %v", err)
	}
	if err := ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}

	deathSystem.Update(w)
	if !ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected zero-health entity without death animation to start fading immediately")
	}

	for range 10 {
		fadeSystem.Update(w)
		deathSystem.Update(w)
	}
	if !ecs.IsAlive(w, e) {
		t.Fatal("expected entity to remain alive until fade out fully completes")
	}

	fadeSystem.Update(w)
	deathSystem.Update(w)
	if ecs.IsAlive(w, e) {
		t.Fatal("expected entity to be destroyed after fade out completes")
	}
}

func TestHealthDeathFadeSystemWaitsForDeathAnimation(t *testing.T) {
	w := ecs.NewWorld()
	deathSystem := NewHealthDeathFadeSystem()
	deathSystem.PostAnimationFrames = 3

	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.HealthComponent.Kind(), &component.Health{Initial: 1, Current: 0}); err != nil {
		t.Fatalf("add health: %v", err)
	}
	if err := ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}
	if err := ecs.Add(w, e, component.AnimationComponent.Kind(), &component.Animation{
		Current: "idle",
		Playing: true,
		Defs: map[string]component.AnimationDef{
			"idle":  {Name: "idle", FrameCount: 1, Loop: true},
			"death": {Name: "death", FrameCount: 3, Loop: false},
		},
	}); err != nil {
		t.Fatalf("add animation: %v", err)
	}

	deathSystem.Update(w)
	anim, ok := ecs.Get(w, e, component.AnimationComponent.Kind())
	if !ok || anim == nil {
		t.Fatal("expected animation component to remain present")
	}
	if anim.Current != "death" {
		t.Fatalf("expected death animation to be selected, got %q", anim.Current)
	}
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected fade out to wait until death animation finishes")
	}

	anim.Frame = 2
	anim.Playing = false
	deathSystem.Update(w)
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected post-animation delay before fade out starts")
	}

	deathSystem.Update(w)
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected fade out to keep waiting during post-animation delay")
	}

	deathSystem.Update(w)
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected fade out to wait until the full post-animation delay elapses")
	}

	deathSystem.Update(w)
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected fade out to start only after the post-animation delay has fully elapsed")
	}

	deathSystem.Update(w)
	if !ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected fade out to start after death animation and the post-animation delay finish")
	}
}

func TestHealthDeathFadeSystemSkipsPlayer(t *testing.T) {
	w := ecs.NewWorld()
	deathSystem := NewHealthDeathFadeSystem()

	e := ecs.CreateEntity(w)
	if err := ecs.Add(w, e, component.HealthComponent.Kind(), &component.Health{Initial: 3, Current: 0}); err != nil {
		t.Fatalf("add health: %v", err)
	}
	if err := ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{}); err != nil {
		t.Fatalf("add sprite: %v", err)
	}
	if err := ecs.Add(w, e, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}

	deathSystem.Update(w)
	if ecs.Has(w, e, component.HealthDeathFadeComponent.Kind()) {
		t.Fatal("expected player death flow to remain owned by the player systems")
	}
	if ecs.Has(w, e, component.SpriteFadeOutComponent.Kind()) {
		t.Fatal("expected player to not receive generic death fade out")
	}
	if !ecs.IsAlive(w, e) {
		t.Fatal("expected player entity to remain alive")
	}
}
