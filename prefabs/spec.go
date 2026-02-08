package prefabs

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type CameraSpec struct {
	Name      string        `yaml:"name"`
	Transform TransformSpec `yaml:"transform"`
	Target    string        `yaml:"target"`
	Zoom      float64       `yaml:"zoom"`
}

func LoadCameraSpec() (*CameraSpec, error) {
	data, err := Load("camera.yaml")
	if err != nil {
		return nil, fmt.Errorf("prefabs: load camera.yaml: %w", err)
	}
	var spec CameraSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("prefabs: unmarshal camera.yaml: %w", err)
	}
	return &spec, nil
}

type PlayerSpec struct {
	Name      string        `yaml:"name"`
	MoveSpeed float64       `yaml:"moveSpeed"`
	JumpSpeed float64       `yaml:"jumpSpeed"`
	Transform TransformSpec `yaml:"transform"`
	Sprite    SpriteSpec    `yaml:"sprite"`
	Animation AnimationSpec `yaml:"animation"`
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

type AnimationSpec struct {
	Sheet      string                      `yaml:"sheet"`
	Defs       map[string]AnimationDefSpec `yaml:"defs"`
	Current    string                      `yaml:"current"`
	Frame      int                         `yaml:"frame"`
	FrameTimer int                         `yaml:"frame_timer"`
	Playing    bool                        `yaml:"playing"`
}

type AnimationDefSpec struct {
	Name       string  `yaml:"name"`
	Row        int     `yaml:"row"`
	ColStart   int     `yaml:"col_start"`
	FrameCount int     `yaml:"frame_count"`
	FrameW     int     `yaml:"frame_w"`
	FrameH     int     `yaml:"frame_h"`
	FPS        float64 `yaml:"fps"`
	Loop       bool    `yaml:"loop"`
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
