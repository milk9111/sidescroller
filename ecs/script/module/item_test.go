package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestItemModuleSetItemReference(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	itemReference := &component.ItemReference{Prefab: "prefabs/items/old.json"}
	if err := ecs.Add(w, entity, component.ItemReferenceComponent.Kind(), itemReference); err != nil {
		t.Fatalf("add item reference: %v", err)
	}

	mod := ItemModule().Build(w, nil, entity, entity)
	result, err := mod["set_item_reference"].(*tengo.UserFunction).Value(&tengo.String{Value: " prefabs/items/new.json "})
	if err != nil {
		t.Fatalf("set_item_reference returned error: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("set_item_reference returned %v, want true", result)
	}
	if itemReference.Prefab != "prefabs/items/new.json" {
		t.Fatalf("expected prefab path to be updated, got %q", itemReference.Prefab)
	}
}
