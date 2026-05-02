package scenes

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type stubPlayerState string

func (s stubPlayerState) Name() string                                  { return string(s) }
func (s stubPlayerState) Enter(ctx *component.PlayerStateContext)       {}
func (s stubPlayerState) Exit(ctx *component.PlayerStateContext)        {}
func (s stubPlayerState) HandleInput(ctx *component.PlayerStateContext) {}
func (s stubPlayerState) Update(ctx *component.PlayerStateContext)      {}

func addGameplayTestPlayer(t *testing.T, w *ecs.World, state string, aimSlowFactor float64) {
	t.Helper()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerComponent.Kind(), &component.Player{AimSlowFactor: aimSlowFactor}); err != nil {
		t.Fatalf("add player component: %v", err)
	}
	if err := ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), &component.PlayerStateMachine{State: stubPlayerState(state)}); err != nil {
		t.Fatalf("add player state machine: %v", err)
	}
}

func TestGameSceneGameplayUpdateStepsUsesAimSlowFactor(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "aim", 0.5)

	g := &GameScene{world: w}
	if steps := g.gameplayUpdateSteps(); steps != 0 {
		t.Fatalf("expected first slowed frame to skip gameplay update, got %d", steps)
	}
	if g.gameplayDebt != 0.5 {
		t.Fatalf("expected gameplay debt 0.5 after first slowed frame, got %v", g.gameplayDebt)
	}
	if steps := g.gameplayUpdateSteps(); steps != 1 {
		t.Fatalf("expected second slowed frame to advance one gameplay step, got %d", steps)
	}
	if g.gameplayDebt != 0 {
		t.Fatalf("expected gameplay debt to be consumed after update, got %v", g.gameplayDebt)
	}
}

func TestGameSceneGameplayUpdateStepsResetsOutsideAim(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "idle", 0.5)

	g := &GameScene{world: w, gameplayDebt: 0.5}
	if steps := g.gameplayUpdateSteps(); steps != 1 {
		t.Fatalf("expected non-aim frame to update gameplay once, got %d", steps)
	}
	if g.gameplayDebt != 0 {
		t.Fatalf("expected gameplay debt reset outside aim, got %v", g.gameplayDebt)
	}
}

func TestGameSceneGameplayUpdateStepsIgnoresInvalidAimSlowFactor(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "aim", 0)

	g := &GameScene{world: w}
	if steps := g.gameplayUpdateSteps(); steps != 1 {
		t.Fatalf("expected invalid aim slow factor to fall back to normal speed, got %d", steps)
	}
}
