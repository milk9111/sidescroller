package editorsystem

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorCommandSystem struct{}

func NewEditorCommandSystem() *EditorCommandSystem {
	return &EditorCommandSystem{}
}

func (s *EditorCommandSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok {
		return
	}
	_, focus, _ := focusState(w)
	_, input, ok := rawInputState(w)
	if !ok {
		return
	}
	if focus != nil && focus.SuppressHotkeys {
		return
	}

	if input.Ctrl {
		switch {
		case inpututil.IsKeyJustPressed(ebiten.KeyB):
			session.ActiveTool = editorcomponent.ToolBrush
		case inpututil.IsKeyJustPressed(ebiten.KeyE):
			session.ActiveTool = editorcomponent.ToolErase
		case inpututil.IsKeyJustPressed(ebiten.KeyF):
			session.ActiveTool = editorcomponent.ToolFill
		case inpututil.IsKeyJustPressed(ebiten.KeyL):
			session.ActiveTool = editorcomponent.ToolLine
		case inpututil.IsKeyJustPressed(ebiten.KeyK):
			session.ActiveTool = editorcomponent.ToolSpike
		case inpututil.IsKeyJustPressed(ebiten.KeyZ):
			session.UndoRequested = true
		case inpututil.IsKeyJustPressed(ebiten.KeyS):
			session.SaveRequested = true
		}
	}

	_, actions, _ := actionState(w)
	if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if actions != nil {
			actions.DeleteSelectedEntity = true
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if actions != nil {
			actions.ClearSelections = true
		}
	}

	if !input.Ctrl {
		if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
			session.CurrentLayer--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyE) {
			session.CurrentLayer++
		}
		if actions != nil {
			if inpututil.IsKeyJustPressed(ebiten.KeyN) {
				actions.AddLayer = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyH) {
				actions.ToggleLayerPhysics = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyY) {
				actions.TogglePhysicsHighlight = true
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyT) {
				actions.ToggleAutotile = true
			}
		}
		clampCurrentLayer(w, session)
	}
}

func clampCurrentLayer(w *ecs.World, session *editorcomponent.EditorSession) {
	if session == nil {
		return
	}
	layers := layerEntities(w)
	if len(layers) == 0 {
		session.CurrentLayer = 0
		return
	}
	if session.CurrentLayer < 0 {
		session.CurrentLayer = 0
	}
	if session.CurrentLayer >= len(layers) {
		session.CurrentLayer = len(layers) - 1
	}
}
