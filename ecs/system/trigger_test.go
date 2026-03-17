package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestTriggerSystemEmitsSignalAndDisablesTrigger(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 16, Y: 16}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 32, Height: 32}); err != nil {
		t.Fatalf("add player physics body: %v", err)
	}

	triggerEntity := ecs.CreateEntity(w)
	if err := ecs.Add(w, triggerEntity, component.TransformComponent.Kind(), &component.Transform{}); err != nil {
		t.Fatalf("add trigger transform: %v", err)
	}
	if err := ecs.Add(w, triggerEntity, component.TriggerComponent.Kind(), &component.Trigger{Name: "once", Bounds: component.AABB{W: 32, H: 32}}); err != nil {
		t.Fatalf("add trigger component: %v", err)
	}

	system := NewTriggerSystem()
	system.Update(w)

	queue, ok := ecs.Get(w, triggerEntity, component.ScriptSignalQueueComponent.Kind())
	if !ok || queue == nil {
		t.Fatal("expected trigger to queue a script signal")
	}
	if len(queue.Events) != 1 || queue.Events[0].Name != "on_trigger_entered" {
		t.Fatalf("expected one on_trigger_entered event, got %#v", queue.Events)
	}

	trigger, _ := ecs.Get(w, triggerEntity, component.TriggerComponent.Kind())
	if trigger == nil || !trigger.Disabled {
		t.Fatal("expected trigger to disable itself after firing")
	}

	system.Update(w)
	queue, _ = ecs.Get(w, triggerEntity, component.ScriptSignalQueueComponent.Kind())
	if len(queue.Events) != 1 {
		t.Fatalf("expected disabled trigger to not enqueue again, got %d events", len(queue.Events))
	}
}
