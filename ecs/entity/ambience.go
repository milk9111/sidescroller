package entity

import (
	"fmt"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/prefabs"
)

func NewAmbience(w *ecs.World) (ecs.Entity, error) {
	ambienceSpec, err := prefabs.LoadSpec[prefabs.AmbienceSpec]("ambience.yaml")
	if err != nil {
		return 0, fmt.Errorf("ambience: load ambience spec: %w", err)
	}

	e := ecs.CreateEntity(w)

	audioComp, err := buildAudioComponent(ambienceSpec.Audio)
	if err != nil {
		return 0, fmt.Errorf("ambience: build audio component: %w", err)
	}
	if audioComp != nil {
		if err := ecs.Add(w, e, component.AudioComponent.Kind(), audioComp); err != nil {
			return 0, fmt.Errorf("ambience: add audio: %w", err)
		}
	}

	audioComp.Play[0] = true

	return e, nil
}
