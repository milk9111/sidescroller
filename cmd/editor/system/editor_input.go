package editorsystem

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorInputSystem struct{}

func NewEditorInputSystem() *EditorInputSystem {
	return &EditorInputSystem{}
}

func (s *EditorInputSystem) Update(w *ecs.World) {
	_, input, ok := rawInputState(w)
	if !ok {
		return
	}
	_, pointer, ok := pointerState(w)
	if !ok {
		return
	}
	_, camera, ok := cameraState(w)
	if !ok {
		return
	}
	_, meta, _ := levelMetaState(w)
	_, clock, hasClock := clockState(w)

	layoutPanels(camera)

	mouseX, mouseY := ebiten.CursorPosition()
	wheelX, wheelY := ebiten.Wheel()
	input.MouseX = mouseX
	input.MouseY = mouseY
	input.WheelX = wheelX
	input.WheelY = wheelY
	input.Ctrl = ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	input.Shift = ebiten.IsKeyPressed(ebiten.KeyShift)
	input.LeftDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	input.RightDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight)
	input.MiddleDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle)
	input.LeftJustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	input.LeftJustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	input.RightJustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	input.RightJustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight)
	input.MiddleJustPressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle)
	input.MiddleJustReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonMiddle)

	pointer.OverLeftPanel = float64(mouseX) < LeftPanelWidth
	pointer.OverRightPanel = float64(mouseX) >= camera.ScreenW-RightPanelWidth
	pointer.OverToolbar = float64(mouseY) < TopToolbarHeight
	pointer.InCanvas = !pointer.OverLeftPanel && !pointer.OverRightPanel && !pointer.OverToolbar &&
		float64(mouseX) >= camera.CanvasX && float64(mouseY) >= camera.CanvasY &&
		float64(mouseX) < camera.CanvasX+camera.CanvasW && float64(mouseY) < camera.CanvasY+camera.CanvasH

	pointer.WorldX = camera.X + (float64(mouseX)-camera.CanvasX)/camera.Zoom
	pointer.WorldY = camera.Y + (float64(mouseY)-camera.CanvasY)/camera.Zoom
	pointer.CellX = int(math.Floor(pointer.WorldX / TileSize))
	pointer.CellY = int(math.Floor(pointer.WorldY / TileSize))
	pointer.HasCell = pointer.InCanvas && withinLevel(meta, pointer.CellX, pointer.CellY)

	if hasClock && clock != nil {
		clock.Frame++
	}
}

func clockState(w *ecs.World) (ecs.Entity, *editorcomponent.EditorClock, bool) {
	entity, ok := ecs.First(w, editorcomponent.EditorClockComponent.Kind())
	if !ok {
		return 0, nil, false
	}
	clock, ok := ecs.Get(w, entity, editorcomponent.EditorClockComponent.Kind())
	return entity, clock, ok && clock != nil
}
