package module

import (
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	levelentity "github.com/milk9111/sidescroller/ecs/entity"
)

func TestDebugModulePrintUpdatesDebugMessageEntity(t *testing.T) {
	w := ecs.NewWorld()
	if _, err := levelentity.NewDebugMessage(w); err != nil {
		t.Fatalf("new debug message: %v", err)
	}

	mod := DebugModule().Build(w, nil, 0, 0)
	printFn, ok := mod["print"].(*tengo.UserFunction)
	if !ok || printFn == nil {
		t.Fatal("expected debug print function")
	}

	result, err := printFn.Value(&tengo.Int{Value: 320}, &tengo.Int{Value: 48}, &tengo.String{Value: "Press C to use the healing flask."})
	if err != nil {
		t.Fatalf("debug print: %v", err)
	}
	if result != tengo.TrueValue {
		t.Fatalf("expected true result, got %#v", result)
	}

	ent, ok := ecs.First(w, component.DebugMessageComponent.Kind())
	if !ok {
		t.Fatal("expected debug message entity")
	}

	debugMessage, _ := ecs.Get(w, ent, component.DebugMessageComponent.Kind())
	if debugMessage == nil || debugMessage.Message != "Press C to use the healing flask." || debugMessage.Width != 320 || debugMessage.Height != 48 {
		t.Fatalf("unexpected debug message state: %+v", debugMessage)
	}

	sprite, _ := ecs.Get(w, ent, component.SpriteComponent.Kind())
	if sprite == nil || sprite.Disabled || sprite.Image == nil {
		t.Fatalf("expected visible debug sprite, got %+v", sprite)
	}
}
