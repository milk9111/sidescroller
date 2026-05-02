package system

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestCameraSystemSnapsOnNewLoadSequence(t *testing.T) {
	w := ecs.NewWorld()

	player := ecs.CreateEntity(w)
	if err := ecs.Add(w, player, component.PlayerTagComponent.Kind(), &component.PlayerTag{}); err != nil {
		t.Fatalf("add player tag: %v", err)
	}
	if err := ecs.Add(w, player, component.TransformComponent.Kind(), &component.Transform{X: 100, Y: 200, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add player transform: %v", err)
	}
	if err := ecs.Add(w, player, component.SpriteComponent.Kind(), &component.Sprite{Image: ebiten.NewImage(64, 64)}); err != nil {
		t.Fatalf("add player sprite: %v", err)
	}
	body := cp.NewBody(1, cp.MomentForBox(1, 20, 40))
	body.SetPosition(cp.Vector{X: 100, Y: 200})
	if err := ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Body: body, Width: 20, Height: 40}); err != nil {
		t.Fatalf("add player body: %v", err)
	}

	cam := ecs.CreateEntity(w)
	if err := ecs.Add(w, cam, component.CameraComponent.Kind(), &component.Camera{TargetName: "player", Zoom: 2, Smoothness: 0.1}); err != nil {
		t.Fatalf("add camera component: %v", err)
	}
	if err := ecs.Add(w, cam, component.TransformComponent.Kind(), &component.Transform{X: 12, Y: 34, ScaleX: 1, ScaleY: 1}); err != nil {
		t.Fatalf("add camera transform: %v", err)
	}

	loaded := ecs.CreateEntity(w)
	if err := ecs.Add(w, loaded, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{Sequence: 1}); err != nil {
		t.Fatalf("add level loaded: %v", err)
	}

	cs := NewCameraSystem()
	cs.SetScreenSize(640, 360)
	cs.Update(w)

	camTransform, _ := ecs.Get(w, cam, component.TransformComponent.Kind())
	firstX, firstY := camTransform.X, camTransform.Y

	camTransform.X = -999
	camTransform.Y = -999
	playerTransform, _ := ecs.Get(w, player, component.TransformComponent.Kind())
	playerTransform.X = 420
	playerTransform.Y = 260
	body.SetPosition(cp.Vector{X: 420, Y: 260})
	loadedComp, _ := ecs.Get(w, loaded, component.LevelLoadedComponent.Kind())
	loadedComp.Sequence = 2

	cs.Update(w)

	if camTransform.X == -999 || camTransform.Y == -999 {
		t.Fatal("expected camera to snap on the new load sequence")
	}
	if camTransform.X == firstX && camTransform.Y == firstY {
		t.Fatal("expected camera snap target to change after player position and load sequence changed")
	}
	if cs.lastLoadSeq != 2 {
		t.Fatalf("expected last load sequence 2, got %d", cs.lastLoadSeq)
	}
}
