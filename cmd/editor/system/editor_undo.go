package editorsystem

import "github.com/milk9111/sidescroller/ecs"

type EditorUndoSystem struct{}

func NewEditorUndoSystem() *EditorUndoSystem {
	return &EditorUndoSystem{}
}

func (s *EditorUndoSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil || !session.UndoRequested {
		return
	}
	defer func() {
		session.UndoRequested = false
	}()

	_, undo, ok := undoState(w)
	if !ok || undo == nil || len(undo.Snapshots) == 0 {
		session.Status = "Undo stack empty"
		return
	}

	last := undo.Snapshots[len(undo.Snapshots)-1]
	undo.Snapshots = undo.Snapshots[:len(undo.Snapshots)-1]
	restoreSnapshot(w, last)
	setDirty(w, true)
}

var _ ecs.System = (*EditorUndoSystem)(nil)
