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

func TestGameSceneGameplayUpdateScaleUsesAimSlowFactor(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "aim", 0.5)

	g := &GameScene{world: w}
	if scale := g.gameplayUpdateScale(); scale != 0.5 {
		t.Fatalf("expected aim slow factor 0.5 while aiming, got %v", scale)
	}
}

func TestGameSceneGameplayUpdateScaleResetsOutsideAim(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "idle", 0.5)

	g := &GameScene{world: w}
	if scale := g.gameplayUpdateScale(); scale != 1 {
		t.Fatalf("expected non-aim frame to use normal gameplay scale, got %v", scale)
	}
}

func TestGameSceneGameplayUpdateScaleIgnoresInvalidAimSlowFactor(t *testing.T) {
	w := ecs.NewWorld()
	addGameplayTestPlayer(t, w, "aim", 0)

	g := &GameScene{world: w}
	if scale := g.gameplayUpdateScale(); scale != 1 {
		t.Fatalf("expected invalid aim slow factor to fall back to normal speed, got %v", scale)
	}
}

func TestGameSceneSetGameplayTimeScaleStoresWorldScale(t *testing.T) {
	w := ecs.NewWorld()
	g := &GameScene{world: w}

	g.setGameplayTimeScale(0.25)

	ent, ok := ecs.First(w, component.GameplayTimeComponent.Kind())
	if !ok {
		t.Fatal("expected gameplay time entity to be created")
	}
	time, ok := ecs.Get(w, ent, component.GameplayTimeComponent.Kind())
	if !ok || time == nil {
		t.Fatal("expected gameplay time component")
	}
	if time.Scale != 0.25 {
		t.Fatalf("expected stored gameplay scale 0.25, got %v", time.Scale)
	}

	g.setGameplayTimeScale(1)
	if time.Scale != 1 {
		t.Fatalf("expected gameplay scale to update in place to 1, got %v", time.Scale)
	}
}
