package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewTransitionPopup(w *ecs.World) (ecs.Entity, error) {
	e, err := BuildEntity(w, "transition_popup.yaml")
	if err != nil {
		return 0, fmt.Errorf("build transition popup: %w", err)
	}

	return e, nil
}
