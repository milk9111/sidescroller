package system

import (
	"testing"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestHandleHealInputQueuesHealWhenFlasksRemain(t *testing.T) {
	state := &component.PlayerStateMachine{State: playerStateIdle, HealUses: playerHealMaxUses - 1}
	input := &component.Input{HealPressed: true}
	abilities := &component.Abilities{Heal: true}
	played := ""

	handleHealInput(input, abilities, state, func(name string) {
		played = name
	})

	if state.Pending != playerStateHeal {
		t.Fatal("expected heal input to queue the heal state when flasks remain")
	}
	if played != "" {
		t.Fatalf("expected no audio when heal can be used, got %q", played)
	}
}

func TestHandleHealInputPlaysOutOfHealingWhenFlasksExhausted(t *testing.T) {
	state := &component.PlayerStateMachine{State: playerStateIdle, HealUses: playerHealMaxUses}
	input := &component.Input{HealPressed: true}
	abilities := &component.Abilities{Heal: true}
	played := ""

	handleHealInput(input, abilities, state, func(name string) {
		played = name
	})

	if state.Pending != nil {
		t.Fatal("expected no heal state to be queued when flasks are exhausted")
	}
	if played != "out_of_healing" {
		t.Fatalf("expected out_of_healing audio, got %q", played)
	}
}

func TestSetPhysicsBodyCenterUpdatesTransform(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	transform := &component.Transform{X: 50, Y: 60, ScaleX: 1, ScaleY: 1}
	body := cp.NewBody(1, cp.MomentForBox(1, 20, 40))
	bodyComp := &component.PhysicsBody{Body: body, Width: 20, Height: 40, OffsetX: 30, OffsetY: 40}

	setPhysicsBodyCenter(w, e, transform, bodyComp, 130, 90)

	pos := body.Position()
	if pos.X != 130 || pos.Y != 90 {
		t.Fatalf("expected physics body center to be updated to (130,90), got (%v,%v)", pos.X, pos.Y)
	}
	if transform.X != 100 || transform.Y != 50 {
		t.Fatalf("expected transform to stay in sync at (100,50), got (%v,%v)", transform.X, transform.Y)
	}
}

func TestDisableAndRestorePlayerCollisions(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	state := &component.PlayerStateMachine{}
	layer := &component.CollisionLayer{Category: 2, Mask: 1}
	if err := ecs.Add(w, e, component.CollisionLayerComponent.Kind(), layer); err != nil {
		t.Fatalf("add collision layer: %v", err)
	}

	disablePlayerCollisions(w, e, state)
	if !state.ClamberCollisionSaved {
		t.Fatal("expected original collision layer to be saved")
	}
	if layer.Category != playerNoClipLayer || layer.Mask != playerNoClipLayer {
		t.Fatalf("expected player collisions to be disabled via noclip layer, got category=%d mask=%d", layer.Category, layer.Mask)
	}

	restorePlayerCollisions(w, e, state)
	if state.ClamberCollisionSaved {
		t.Fatal("expected saved collision layer to be cleared after restore")
	}
	if layer.Category != 2 || layer.Mask != 1 {
		t.Fatalf("expected original collision layer to be restored, got category=%d mask=%d", layer.Category, layer.Mask)
	}
}
