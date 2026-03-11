package editorsystem

import (
	"fmt"

	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorPersistenceSystem struct {
	workspaceRoot string
	levelDir      string
}

func NewEditorPersistenceSystem(workspaceRoot, levelDir string) *EditorPersistenceSystem {
	return &EditorPersistenceSystem{workspaceRoot: workspaceRoot, levelDir: levelDir}
}

func (s *EditorPersistenceSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil || !session.SaveRequested {
		return
	}
	defer func() {
		session.SaveRequested = false
	}()
	if _, entities, ok := entitiesState(w); ok && entities != nil {
		if ensureUniqueEntityIDs(entities.Items) {
			setDirty(w, true)
		}
	}

	doc := cloneCurrentLevel(w)
	normalized, err := editorio.SaveLevel(s.workspaceRoot, s.levelDir, session.SaveTarget, &doc)
	if err != nil {
		session.Status = fmt.Sprintf("Save failed: %v", err)
		return
	}
	session.SaveTarget = normalized
	session.LoadedLevel = normalized
	session.Status = "Saved levels/" + normalized
	setDirty(w, false)
}

var _ ecs.System = (*EditorPersistenceSystem)(nil)
