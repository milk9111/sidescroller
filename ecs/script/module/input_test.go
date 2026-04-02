package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestInputModuleReportsGamepadUsage(t *testing.T) {
	w := ecs.NewWorld()
	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.InputComponent.Kind(), &component.Input{UsingGamepad: true}); err != nil {
		t.Fatalf("add input component: %v", err)
	}

	mod := InputModule().Build(w, nil, player, player)
	fn, ok := mod["is_using_gamepad"].(*tengo.UserFunction)
	if !ok || fn == nil {
		t.Fatal("expected is_using_gamepad function")
	}

	result, err := fn.Value()
	if err != nil {
		t.Fatalf("is_using_gamepad: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("expected true result, got %#v", result)
	}

	input, _ := ecs.Get(w, player, component.InputComponent.Kind())
	input.UsingGamepad = false

	result, err = fn.Value()
	if err != nil {
		t.Fatalf("is_using_gamepad second call: %v", err)
	}
	if result != tengo.FalseValue {
		t.Fatalf("expected false result, got %#v", result)
	}
}
