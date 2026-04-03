package system

import (
	"testing"

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
