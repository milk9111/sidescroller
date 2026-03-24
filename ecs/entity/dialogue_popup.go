package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewDialoguePopup(w *ecs.World) (ecs.Entity, error) {
	e, err := BuildEntity(w, "dialogue_popup.yaml")
	if err != nil {
		return 0, fmt.Errorf("build dialogue popup: %w", err)
	}

	return e, nil
}
