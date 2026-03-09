package module

import (
	"math"
	"testing"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestTransformModuleRotationUsesLocalRotationForUnparentedEntity(t *testing.T) {
	w := ecs.NewWorld()
	entity := ecs.CreateEntity(w)
	if err := ecs.Add(w, entity, component.TransformComponent.Kind(), &component.Transform{
		Rotation:      0,
		WorldRotation: math.Pi / 2,
		ScaleX:        1,
		ScaleY:        1,
	}); err != nil {
		t.Fatalf("add transform: %v", err)
	}

	mod := TransformModule().Build(w, nil, entity, entity)
	rotationObj, err := mod["rotation"].(*tengo.UserFunction).Value()
	if err != nil {
		t.Fatalf("rotation returned error: %v", err)
	}
	rotation, ok := rotationObj.(*tengo.Float)
	if !ok {
		t.Fatalf("rotation returned %T, want *tengo.Float", rotationObj)
	}
	if rotation.Value != 0 {
		t.Fatalf("expected local rotation 0 degrees, got %v", rotation.Value)
	}
}
