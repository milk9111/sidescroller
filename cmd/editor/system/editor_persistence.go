package editorsystem

import (
	"fmt"

	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorPersistenceSystem struct {
	workspaceRoot string
}

func NewEditorPersistenceSystem(workspaceRoot string) *EditorPersistenceSystem {
	return &EditorPersistenceSystem{workspaceRoot: workspaceRoot}
}

func (s *EditorPersistenceSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil || !session.SaveRequested {
		return
	}
	defer func() {
		session.SaveRequested = false
	}()

	doc := cloneCurrentLevel(w)
	normalized, err := editorio.SaveLevel(s.workspaceRoot, session.SaveTarget, &doc)
	if err != nil {
		session.Status = fmt.Sprintf("Save failed: %v", err)
		return
	}
	session.SaveTarget = normalized
	session.LoadedLevel = normalized
	session.Status = "Saved " + normalized
	setDirty(w, false)
}

var _ ecs.System = (*EditorPersistenceSystem)(nil)
