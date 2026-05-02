package system

import (
	"math"
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

func TestShouldClamberRequiresMovementIntoWall(t *testing.T) {
	ctx := &component.PlayerStateContext{
		Input:      &component.Input{MoveX: 1},
		CanClamber: func() bool { return true },
		GetPosition: func() (x, y float64) {
			return 10, 0
		},
		GetClamberTarget: func() (x, y float64) {
			return 30, 0
		},
		WallSide:   func() int { return wallRight },
		IsGrounded: func() bool { return false },
	}
	if !shouldClamber(ctx) {
		t.Fatal("expected clamber when moving into a detected right ledge")
	}

	ctx.Input.MoveX = -1
	if shouldClamber(ctx) {
		t.Fatal("expected clamber to require movement toward the clamber target")
	}
	ctx.Input.MoveX = -1
	ctx.GetClamberTarget = func() (x, y float64) {
		return -10, 0
	}
	if !shouldClamber(ctx) {
		t.Fatal("expected clamber when moving left toward a left-side clamber target")
	}
	ctx.Input.MoveX = 1
	ctx.IsGrounded = func() bool { return true }
	if shouldClamber(ctx) {
		t.Fatal("expected grounded players to skip clamber")
	}
}

func TestPlayerClamberStateMovesPlayerToTarget(t *testing.T) {
	state := &component.PlayerStateMachine{}
	positionX := 10.0
	positionY := 80.0
	changedTo := ""
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{ClamberFrames: 4},
		GetPosition: func() (x, y float64) {
			return positionX, positionY
		},
		SetPosition: func(x, y float64) {
			positionX = x
			positionY = y
		},
		SetVelocity:     func(x, y float64) {},
		SetGravityScale: func(scale float64) {},
		CanClamber:      func() bool { return true },
		GetClamberTarget: func() (x, y float64) {
			return 30, 40
		},
		GetClamberFrames: func() int {
			return state.ClamberFramesElapsed
		},
		SetClamberFrames: func(frames int) {
			state.ClamberFramesElapsed = frames
		},
		GetClamberStart: func() (x, y float64) {
			return state.ClamberStartX, state.ClamberStartY
		},
		SetClamberStart: func(x, y float64) {
			state.ClamberStartX = x
			state.ClamberStartY = y
		},
		GetStoredClamberTarget: func() (x, y float64) {
			return state.ClamberTargetX, state.ClamberTargetY
		},
		SetStoredClamberTarget: func(x, y float64) {
			state.ClamberTargetX = x
			state.ClamberTargetY = y
		},
		GetAnimationDuration: func(animation string) int {
			if animation == "clamber" {
				return 4
			}
			return 0
		},
		WallSide:        func() int { return wallRight },
		FacingLeft:      func(bool) {},
		ChangeAnimation: func(string) {},
		ChangeState: func(next component.PlayerState) {
			if next != nil {
				changedTo = next.Name()
			}
		},
	}

	playerStateClamber.Enter(ctx)
	for i := 0; i < 4; i++ {
		playerStateClamber.Update(ctx)
	}

	if changedTo != "idle" {
		t.Fatalf("expected clamber to end in idle, got %q", changedTo)
	}
	if math.Abs(positionX-30) > 0.001 || math.Abs(positionY-40) > 0.001 {
		t.Fatalf("expected clamber to finish at target (30,40), got (%v,%v)", positionX, positionY)
	}
	if state.ClamberFramesElapsed != 4 {
		t.Fatalf("expected clamber frames to advance to 4, got %d", state.ClamberFramesElapsed)
	}
}

func TestPlayerClamberStateWaitsForAnimationToFinish(t *testing.T) {
	state := &component.PlayerStateMachine{}
	positionX := 10.0
	positionY := 80.0
	changedTo := ""
	animationPlaying := true
	setPositionCalls := 0
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{ClamberFrames: 2},
		GetPosition: func() (x, y float64) {
			return positionX, positionY
		},
		SetPosition: func(x, y float64) {
			setPositionCalls++
			positionX = x
			positionY = y
		},
		SetVelocity:     func(x, y float64) {},
		SetGravityScale: func(scale float64) {},
		CanClamber:      func() bool { return true },
		GetClamberTarget: func() (x, y float64) {
			return 30, 40
		},
		GetClamberFrames: func() int {
			return state.ClamberFramesElapsed
		},
		SetClamberFrames: func(frames int) {
			state.ClamberFramesElapsed = frames
		},
		GetClamberStart: func() (x, y float64) {
			return state.ClamberStartX, state.ClamberStartY
		},
		SetClamberStart: func(x, y float64) {
			state.ClamberStartX = x
			state.ClamberStartY = y
		},
		GetStoredClamberTarget: func() (x, y float64) {
			return state.ClamberTargetX, state.ClamberTargetY
		},
		SetStoredClamberTarget: func(x, y float64) {
			state.ClamberTargetX = x
			state.ClamberTargetY = y
		},
		GetAnimationDuration: func(animation string) int {
			if animation == "clamber" {
				return 2
			}
			return 0
		},
		GetAnimationPlaying: func() bool {
			return animationPlaying
		},
		FacingLeft:      func(bool) {},
		ChangeAnimation: func(string) {},
		ChangeState: func(next component.PlayerState) {
			if next != nil {
				changedTo = next.Name()
			}
		},
	}

	playerStateClamber.Enter(ctx)
	playerStateClamber.Update(ctx)
	playerStateClamber.Update(ctx)

	if changedTo != "" {
		t.Fatalf("expected clamber to hold until animation finishes, got transition %q", changedTo)
	}
	if setPositionCalls != 2 {
		t.Fatalf("expected clamber to position during movement and once on completion, got %d position updates", setPositionCalls)
	}

	animationPlaying = false
	playerStateClamber.Update(ctx)

	if changedTo != "idle" {
		t.Fatalf("expected clamber to exit to idle after animation finishes, got %q", changedTo)
	}
}

func TestPlayerClamberStateUsesAnimationDurationWhenLonger(t *testing.T) {
	state := &component.PlayerStateMachine{}
	positionX := 10.0
	positionY := 80.0
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{ClamberFrames: 4},
		GetPosition: func() (x, y float64) {
			return positionX, positionY
		},
		SetPosition: func(x, y float64) {
			positionX = x
			positionY = y
		},
		SetVelocity:     func(x, y float64) {},
		SetGravityScale: func(scale float64) {},
		CanClamber:      func() bool { return true },
		GetClamberTarget: func() (x, y float64) {
			return 30, 40
		},
		GetClamberFrames: func() int {
			return state.ClamberFramesElapsed
		},
		SetClamberFrames: func(frames int) {
			state.ClamberFramesElapsed = frames
		},
		GetClamberStart: func() (x, y float64) {
			return state.ClamberStartX, state.ClamberStartY
		},
		SetClamberStart: func(x, y float64) {
			state.ClamberStartX = x
			state.ClamberStartY = y
		},
		GetStoredClamberTarget: func() (x, y float64) {
			return state.ClamberTargetX, state.ClamberTargetY
		},
		SetStoredClamberTarget: func(x, y float64) {
			state.ClamberTargetX = x
			state.ClamberTargetY = y
		},
		GetAnimationDuration: func(animation string) int {
			if animation == "clamber" {
				return 10
			}
			return 0
		},
		GetAnimationPlaying: func() bool { return true },
		FacingLeft:          func(bool) {},
		ChangeAnimation:     func(string) {},
		ChangeState:         func(component.PlayerState) {},
	}

	playerStateClamber.Enter(ctx)
	for i := 0; i < 4; i++ {
		playerStateClamber.Update(ctx)
	}

	if math.Abs(positionX-18) > 0.001 || math.Abs(positionY-64) > 0.001 {
		t.Fatalf("expected clamber motion to use the longer animation duration, got (%v,%v)", positionX, positionY)
	}
}

func TestPlayerShrineHealStateCompletesShrineEffectsAfterAnimation(t *testing.T) {
	completed := false
	changedTo := ""
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{},
		Player: &component.Player{MoveSpeed: 1},
		GetVelocity: func() (x, y float64) {
			return 3, -1
		},
		SetVelocity: func(x, y float64) {},
		IsGrounded: func() bool {
			return true
		},
		GetAnimationPlaying: func() bool {
			return false
		},
		ChangeAnimation: func(name string) {
			if name != "shrine_heal" {
				t.Fatalf("expected shrine_heal animation, got %q", name)
			}
		},
		CompleteShrineHeal: func() {
			completed = true
		},
		ChangeState: func(next component.PlayerState) {
			if next != nil {
				changedTo = next.Name()
			}
		},
	}

	playerStateShrine.Enter(ctx)
	playerStateShrine.Update(ctx)

	if !completed {
		t.Fatal("expected shrine state to complete shrine effects when animation ends")
	}
	if changedTo != "idle" {
		t.Fatalf("expected shrine state to return to idle, got %q", changedTo)
	}
}

func TestPlayerShrineHealStateIgnoresInputUntilAnimationFinishes(t *testing.T) {
	completed := false
	changedTo := ""
	ctx := &component.PlayerStateContext{
		Input:  &component.Input{MoveX: 1, JumpPressed: true, AttackPressed: true},
		Player: &component.Player{MoveSpeed: 1},
		GetVelocity: func() (x, y float64) {
			return 3, 0
		},
		SetVelocity: func(x, y float64) {},
		GetAnimationPlaying: func() bool {
			return true
		},
		ChangeAnimation: func(string) {},
		CompleteShrineHeal: func() {
			completed = true
		},
		ChangeState: func(next component.PlayerState) {
			if next != nil {
				changedTo = next.Name()
			}
		},
	}

	playerStateShrine.Enter(ctx)
	playerStateShrine.HandleInput(ctx)
	playerStateShrine.Update(ctx)

	if completed {
		t.Fatal("expected shrine state to wait for animation completion before applying effects")
	}
	if changedTo != "" {
		t.Fatalf("expected shrine state to ignore input while animation is playing, got transition %q", changedTo)
	}
}
