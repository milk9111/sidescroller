package prefabs

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type TransitionSpec struct {
	Name        string          `yaml:"name"`
	Transform   TransformSpec   `yaml:"transform"`
	Sprite      SpriteSpec      `yaml:"sprite"`
	RenderLayer RenderLayerSpec `yaml:"render_layer"`
	Animation   AnimationSpec   `yaml:"animation"`
	Audio       []AudioSpec     `yaml:"audio"`
}

func LoadSpec[T any](filename string) (T, error) {
	var zero T
	data, err := Load(filename)
	if err != nil {
		return zero, fmt.Errorf("prefabs: load %s: %w", filename, err)
	}

	var spec T
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return zero, fmt.Errorf("prefabs: unmarshal %s: %w", filename, err)
	}

	return spec, nil
}

type AnchorSpec struct {
	Name        string          `yaml:"name"`
	Speed       float64         `yaml:"speed"`
	Transform   TransformSpec   `yaml:"transform"`
	Sprite      SpriteSpec      `yaml:"sprite"`
	RenderLayer RenderLayerSpec `yaml:"render_layer"`
	Audio       []AudioSpec     `yaml:"audio"`
}

func LoadAnchorSpec() (*AnchorSpec, error) {
	data, err := Load("anchor.yaml")
	if err != nil {
		return nil, fmt.Errorf("prefabs: load anchor.yaml: %w", err)
	}
	var spec AnchorSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("prefabs: unmarshal anchor.yaml: %w", err)
	}
	return &spec, nil
}

type AimTargetSpec struct {
	Name        string          `yaml:"name"`
	Transform   TransformSpec   `yaml:"transform"`
	Sprite      SpriteSpec      `yaml:"sprite"`
	RenderLayer RenderLayerSpec `yaml:"render_layer"`
	LineRender  LineRenderSpec  `yaml:"line_render"`
	Audio       []AudioSpec     `yaml:"audio"`
}

func LoadAimTargetSpec() (*AimTargetSpec, error) {
	data, err := Load("aim_target.yaml")
	if err != nil {
		return nil, fmt.Errorf("prefabs: load aim_target.yaml: %w", err)
	}
	var spec AimTargetSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("prefabs: unmarshal aim_target.yaml: %w", err)
	}
	return &spec, nil
}

type CameraSpec struct {
	Name       string        `yaml:"name"`
	Transform  TransformSpec `yaml:"transform"`
	Target     string        `yaml:"target"`
	Zoom       float64       `yaml:"zoom"`
	Smoothness float64       `yaml:"smoothness"`
	Audio      []AudioSpec   `yaml:"audio"`
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
	Name             string          `yaml:"name"`
	MoveSpeed        float64         `yaml:"move_speed"`
	JumpSpeed        float64         `yaml:"jump_speed"`
	JumpHoldFrames   int             `yaml:"jump_hold_frames"`
	JumpHoldBoost    float64         `yaml:"jump_hold_boost"`
	CoyoteFrames     int             `yaml:"coyote_frames"`
	JumpBufferFrames int             `yaml:"jump_buffer_frames"`
	WallGrabFrames   int             `yaml:"wall_grab_frames"`
	WallSlideSpeed   float64         `yaml:"wall_slide_speed"`
	WallJumpPush     float64         `yaml:"wall_jump_push"`
	WallJumpFrames   int             `yaml:"wall_jump_frames"`
	AimSlowFactor    float64         `yaml:"aim_slow_factor"`
	Transform        TransformSpec   `yaml:"transform"`
	Collider         ColliderSpec    `yaml:"collider"`
	Sprite           SpriteSpec      `yaml:"sprite"`
	Animation        AnimationSpec   `yaml:"animation"`
	RenderLayer      RenderLayerSpec `yaml:"render_layer"`
	Audio            []AudioSpec     `yaml:"audio"`
	Health           int             `yaml:"health"`
	Hitboxes         []HitboxSpec    `yaml:"hitboxes"`
	Hurtboxes        []HurtboxSpec   `yaml:"hurtboxes"`
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

type EnemySpec struct {
	Name         string          `yaml:"name"`
	MoveSpeed    float64         `yaml:"move_speed"`
	FollowRange  float64         `yaml:"follow_range"`
	AttackRange  float64         `yaml:"attack_range"`
	AttackFrames int             `yaml:"attack_frames"`
	FSM          FSMSpec         `yaml:"fsm"`
	Transform    TransformSpec   `yaml:"transform"`
	Collider     ColliderSpec    `yaml:"collider"`
	Sprite       SpriteSpec      `yaml:"sprite"`
	Animation    AnimationSpec   `yaml:"animation"`
	RenderLayer  RenderLayerSpec `yaml:"render_layer"`
	Audio        []AudioSpec     `yaml:"audio"`
	Health       int             `yaml:"health"`
	Hitboxes     []HitboxSpec    `yaml:"hitboxes"`
	Hurtboxes    []HurtboxSpec   `yaml:"hurtboxes"`
}

type AudioSpec struct {
	Name   string  `yaml:"name"`
	File   string  `yaml:"file"`
	Volume float64 `yaml:"volume"`
}

func LoadEnemySpec() (*EnemySpec, error) {
	data, err := Load("enemy.yaml")
	if err != nil {
		return nil, fmt.Errorf("prefabs: load enemy.yaml: %w", err)
	}
	var spec EnemySpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("prefabs: unmarshal enemy.yaml: %w", err)
	}
	return &spec, nil
}

type FSMSpec struct {
	Initial     string                         `yaml:"initial"`
	States      map[string]FSMStateSpec        `yaml:"states"`
	Transitions map[string][]map[string]string `yaml:"transitions"`
}

type FSMStateSpec struct {
	OnEnter []map[string]any `yaml:"on_enter"`
	While   []map[string]any `yaml:"while"`
	OnExit  []map[string]any `yaml:"on_exit"`
}

type LineRenderSpec struct {
	StartX    float64    `yaml:"start_x"`
	StartY    float64    `yaml:"start_y"`
	EndX      float64    `yaml:"end_x"`
	EndY      float64    `yaml:"end_y"`
	Width     float32    `yaml:"width"`
	Color     *YAMLColor `yaml:"color"`
	AntiAlias bool       `yaml:"anti_alias"`
}

type RenderLayerSpec struct {
	Index int `yaml:"index"`
}

type TransformSpec struct {
	X        float64 `yaml:"x"`
	Y        float64 `yaml:"y"`
	ScaleX   float64 `yaml:"scale_x"`
	ScaleY   float64 `yaml:"scale_y"`
	Rotation float64 `yaml:"rotation"`
}

type ColliderSpec struct {
	Width   float64 `yaml:"width"`
	Height  float64 `yaml:"height"`
	OffsetX float64 `yaml:"offsetX"`
	OffsetY float64 `yaml:"offsetY"`
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

type HitboxSpec struct {
	Width   float64 `yaml:"width"`
	Height  float64 `yaml:"height"`
	OffsetX float64 `yaml:"offset_x"`
	OffsetY float64 `yaml:"offset_y"`
	Damage  int     `yaml:"damage"`
	Anim    string  `yaml:"anim"`
	Frames  []int   `yaml:"frames"`
}

type HurtboxSpec struct {
	Width   float64 `yaml:"width"`
	Height  float64 `yaml:"height"`
	OffsetX float64 `yaml:"offset_x"`
	OffsetY float64 `yaml:"offset_y"`
}

type YAMLColor struct {
	color.Color
}

func (c *YAMLColor) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("color must be a string")
	}

	s := strings.TrimPrefix(value.Value, "#")

	if len(s) != 6 && len(s) != 8 {
		return fmt.Errorf("invalid color format: %s", value.Value)
	}

	parse := func(start int) (uint8, error) {
		v, err := strconv.ParseUint(s[start:start+2], 16, 8)
		return uint8(v), err
	}

	r, err := parse(0)
	if err != nil {
		return err
	}
	g, err := parse(2)
	if err != nil {
		return err
	}
	b, err := parse(4)
	if err != nil {
		return err
	}

	a := uint8(255)
	if len(s) == 8 {
		a, err = parse(6)
		if err != nil {
			return err
		}
	}

	c.Color = color.NRGBA{R: r, G: g, B: b, A: a}
	return nil
}
