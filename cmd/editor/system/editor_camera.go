package editorsystem

import (
	"math"

	"github.com/milk9111/sidescroller/ecs"
)

type EditorCameraSystem struct{}

func NewEditorCameraSystem() *EditorCameraSystem {
	return &EditorCameraSystem{}
}

func (s *EditorCameraSystem) Update(w *ecs.World) {
	if _, session, ok := sessionState(w); ok && session != nil && session.OverviewOpen {
		return
	}
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

	if camera.Zoom <= 0 {
		camera.Zoom = 1
	}

	if input.MiddleDown && pointer.InCanvas && !camera.PanActive {
		camera.PanActive = true
		camera.PanMouseX = input.MouseX
		camera.PanMouseY = input.MouseY
		camera.PanStartX = camera.X
		camera.PanStartY = camera.Y
	}
	if input.MiddleJustReleased || !input.MiddleDown {
		camera.PanActive = false
	}
	if camera.PanActive && input.MiddleDown {
		dx := float64(input.MouseX-camera.PanMouseX) / camera.Zoom
		dy := float64(input.MouseY-camera.PanMouseY) / camera.Zoom
		camera.X = camera.PanStartX - dx
		camera.Y = camera.PanStartY - dy
	}

	if pointer.InCanvas && input.WheelY != 0 {
		beforeWorldX := pointer.WorldX
		beforeWorldY := pointer.WorldY
		camera.Zoom = clampFloat(camera.Zoom+(input.WheelY*0.125), 0.25, 4.0)
		camera.X = beforeWorldX - (float64(input.MouseX)-camera.CanvasX)/camera.Zoom
		camera.Y = beforeWorldY - (float64(input.MouseY)-camera.CanvasY)/camera.Zoom
	}

	camera.X = math.Max(camera.X, -float64(TileSize)*2)
	camera.Y = math.Max(camera.Y, -float64(TileSize)*2)
}

func clampFloat(value, minValue, maxValue float64) float64 {
	return math.Min(maxValue, math.Max(minValue, value))
}

var _ ecs.System = (*EditorCameraSystem)(nil)
