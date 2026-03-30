package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
)

func NewItemPopup(w *ecs.World) (ecs.Entity, error) {
	e, err := BuildEntity(w, "item_popup.yaml")
	if err != nil {
		return 0, fmt.Errorf("build item popup: %w", err)
	}

	return e, nil
}
