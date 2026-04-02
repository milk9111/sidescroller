package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs/component"
)

func TestPlayerFallStateSuppressesLandingAudioForMicroFalls(t *testing.T) {
	state := &component.PlayerStateMachine{}
	playedLand := false
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{MoveSpeed: 1},
		GetVelocity: func() (x, y float64) {
			return 0, 0.5
		},
		SetVelocity: func(x, y float64) {},
		IsGrounded: func() bool {
			return true
		},
		GetFallFrames: func() int {
			return state.FallFrames
		},
		SetFallFrames: func(frames int) {
			state.FallFrames = frames
		},
		ChangeAnimation: func(string) {},
		ChangeState:     func(component.PlayerState) {},
		FacingLeft:      func(bool) {},
		PlayAudio: func(name string) {
			if name == "land" {
				playedLand = true
			}
		},
	}

	playerStateFall.Enter(ctx)
	playerStateFall.Update(ctx)

	if playedLand {
		t.Fatal("expected micro-fall landing audio to be suppressed")
	}
	if state.FallFrames != 1 {
		t.Fatalf("expected fall frames to increment to 1, got %d", state.FallFrames)
	}
}

func TestPlayerFallStatePlaysLandingAudioAfterRealFall(t *testing.T) {
	state := &component.PlayerStateMachine{FallFrames: minLandingSoundFallFrames - 1}
	playedLand := false
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{MoveSpeed: 1},
		GetVelocity: func() (x, y float64) {
			return 0, 3
		},
		SetVelocity: func(x, y float64) {},
		IsGrounded: func() bool {
			return true
		},
		GetFallFrames: func() int {
			return state.FallFrames
		},
		SetFallFrames: func(frames int) {
			state.FallFrames = frames
		},
		ChangeAnimation: func(string) {},
		ChangeState:     func(component.PlayerState) {},
		FacingLeft:      func(bool) {},
		PlayAudio: func(name string) {
			if name == "land" {
				playedLand = true
			}
		},
	}

	playerStateFall.Update(ctx)

	if !playedLand {
		t.Fatal("expected landing audio after a non-trivial fall")
	}
	if state.FallFrames != minLandingSoundFallFrames {
		t.Fatalf("expected fall frames to increment to %d, got %d", minLandingSoundFallFrames, state.FallFrames)
	}
}
