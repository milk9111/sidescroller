package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestScriptModuleAddPathCreatesScriptComponent(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)

	mod := ScriptModule().Build(w, nil, entity, entity)
	result, err := mod["add_path"].(*tengo.UserFunction).Value(&tengo.String{Value: "prefabs/scripts/claw_pickup.tengo"})
	if err != nil {
		t.Fatalf("add_path returned error: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("add_path returned %v, want true", result)
	}

	scriptComp, ok := ecs.Get(w, entity, component.ScriptComponent.Kind())
	if !ok || scriptComp == nil {
		t.Fatal("expected script component to be added")
	}
	if len(scriptComp.Paths) != 1 || scriptComp.Paths[0] != "claw_pickup.tengo" {
		t.Fatalf("expected paths to contain claw_pickup.tengo, got %#v", scriptComp.Paths)
	}
}

func TestScriptModuleAddPathUpgradesLegacyPathAndResetsRuntime(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	if err := ecs.Add(w, entity, component.ScriptComponent.Kind(), &component.Script{Path: "enemy/main.tengo"}); err != nil {
		t.Fatalf("add script component: %v", err)
	}
	if err := ecs.Add(w, entity, component.ScriptRuntimeComponent.Kind(), &component.ScriptRuntime{Started: true}); err != nil {
		t.Fatalf("add script runtime component: %v", err)
	}

	mod := ScriptModule().Build(w, nil, entity, entity)
	result, err := mod["add_path"].(*tengo.UserFunction).Value(&tengo.String{Value: "forsaken_scion/death_state.tengo"})
	if err != nil {
		t.Fatalf("add_path returned error: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("add_path returned %v, want true", result)
	}

	scriptComp, _ := ecs.Get(w, entity, component.ScriptComponent.Kind())
	if got, want := len(scriptComp.Paths), 2; got != want {
		t.Fatalf("expected %d script paths, got %d (%#v)", want, got, scriptComp.Paths)
	}
	if scriptComp.Paths[0] != "enemy/main.tengo" {
		t.Fatalf("expected legacy path to be preserved first, got %#v", scriptComp.Paths)
	}
	if scriptComp.Paths[1] != "forsaken_scion/death_state.tengo" {
		t.Fatalf("expected new path to be appended, got %#v", scriptComp.Paths)
	}

	runtimeComp, _ := ecs.Get(w, entity, component.ScriptRuntimeComponent.Kind())
	if runtimeComp == nil || runtimeComp.Started {
		t.Fatalf("expected script runtime to be reset, got %+v", runtimeComp)
	}
	if scriptComp.Path != "enemy/main.tengo" {
		t.Fatalf("expected legacy path field to remain unchanged, got %q", scriptComp.Path)
	}
}
