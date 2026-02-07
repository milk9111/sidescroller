package prefabs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type PlayerSpec struct {
	Name      string        `yaml:"name"`
	Transform TransformSpec `yaml:"transform"`
	Sprite    SpriteSpec    `yaml:"sprite"`
}

type TransformSpec struct {
	X        float64 `yaml:"x"`
	Y        float64 `yaml:"y"`
	ScaleX   float64 `yaml:"scale_x"`
	ScaleY   float64 `yaml:"scale_y"`
	Rotation float64 `yaml:"rotation"`
}

type SpriteSpec struct {
	Image     string  `yaml:"image"`
	UseSource bool    `yaml:"use_source"`
	OriginX   float64 `yaml:"origin_x"`
	OriginY   float64 `yaml:"origin_y"`
}

func LoadPlayerSpec() (*PlayerSpec, error) {
	data, err := Load("player.yaml")
	if err != nil {
		return nil, fmt.Errorf("prefabs: load player.yaml: %w", err)
	}
	var spec PlayerSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("prefabs: unmarshal player.yaml: %w", err)
	}
	return &spec, nil
}
