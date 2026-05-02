package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewShrinePopup(w *ecs.World) (ecs.Entity, error) {
	e, err := BuildEntity(w, "shrine_popup.yaml")
	if err != nil {
		return 0, fmt.Errorf("build shrine popup: %w", err)
	}

	return e, nil
}