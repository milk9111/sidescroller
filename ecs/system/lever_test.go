package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestLeverSystemClosesAndRecordsPersistedState(t *testing.T) {
	w := ecs.NewWorld()
	addTestLevelRuntime(t, w, "disposal_1.json")
	stateMap, _ := addTestPlayerStateMap(t, w)

	listener := ecs.CreateEntity(w)
	if err := ecs.Add(w, listener, component.ScriptComponent.Kind(), &component.Script{Path: "dummy.tengo"}); err != nil {
		t.Fatalf("add listener script component: %v", err)
	}

	lever := ecs.CreateEntity(w)
	if err := ecs.Add(w, lever, component.GameEntityIDComponent.Kind(), &component.GameEntityID{Value: "lever_1"}); err != nil {
		t.Fatalf("add lever game id: %v", err)
	}
	if err := ecs.Add(w, lever, component.TransformComponent.Kind(), &component.Transform{X: 32, Y: 64, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add lever transform: %v", err)
	}
	if err := ecs.Add(w, lever, component.LeverComponent.Kind(), &component.Lever{
		OpenAnimation:    "open",
		ClosingAnimation: "open_to_closed",
		ClosedAnimation:  "closed",
		State:            component.LeverStateOpen,
	}); err != nil {
		t.Fatalf("add lever component: %v", err)
	}
	if err := ecs.Add(w, lever, component.AnimationComponent.Kind(), &component.Animation{Defs: map[string]component.AnimationDef{
		"open":           {FrameCount: 1, Loop: true},
		"open_to_closed": {FrameCount: 4, Loop: false},
		"closed":         {FrameCount: 1, Loop: true},
	}, Current: "open", Playing: true}); err != nil {
		t.Fatalf("add animation component: %v", err)
	}
	if err := ecs.Add(w, lever, component.LeverHitRequestComponent.Kind(), &component.LeverHitRequest{SourceEntity: 1}); err != nil {
		t.Fatalf("add lever hit request: %v", err)
	}

	system := NewLeverSystem()
	system.Update(w)

	leverComp, _ := ecs.Get(w, lever, component.LeverComponent.Kind())
	if leverComp == nil || leverComp.State != component.LeverStateClosing {
		t.Fatalf("expected lever to enter closing state, got %+v", leverComp)
	}
	anim, _ := ecs.Get(w, lever, component.AnimationComponent.Kind())
	if anim == nil || anim.Current != "open_to_closed" || !anim.Playing {
		t.Fatalf("expected closing animation to start, got %+v", anim)
	}
	if _, ok := ecs.Get(w, lever, component.LeverHitRequestComponent.Kind()); ok {
		t.Fatal("expected lever hit request to be consumed")
	}
	if got := stateMap.States[levelEntityStateKey("disposal_1.json", "lever_1")]; got != component.PersistedLevelEntityStateUsed {
		t.Fatalf("expected lever used state to be recorded, got %q", got)
	}

	anim.Frame = 3
	anim.Playing = false
	system.Update(w)

	if leverComp.State != component.LeverStateClosed {
		t.Fatalf("expected lever to finish closed, got %+v", leverComp)
	}
	if anim.Current != "closed" || !anim.Playing {
		t.Fatalf("expected closed animation to hold, got %+v", anim)
	}
	queue, ok := ecs.Get(w, listener, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil || len(queue.Events) != 1 {
		t.Fatalf("expected one lever closed signal to be broadcast, got %+v", queue)
	}
	if queue.Events[0].Name != "on_lever_closed" || queue.Events[0].SourceGameEntity != "lever_1" {
		t.Fatalf("expected broadcast on_lever_closed from lever_1, got %+v", queue.Events[0])
	}
	if !queue.Events[0].HasPosition || queue.Events[0].PositionX != 32 || queue.Events[0].PositionY != 64 {
		t.Fatalf("expected lever closed signal position (32,64), got %+v", queue.Events[0])
	}
}
