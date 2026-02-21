package prefabs

import "gopkg.in/yaml.v3"

type EntityBuildSpec struct {
	Name       string         `yaml:"name"`
	Components map[string]any `yaml:"components"`
}

func LoadEntityBuildSpec(filename string) (EntityBuildSpec, error) {
	return LoadSpec[EntityBuildSpec](filename)
}

func DecodeComponentSpec[T any](raw any) (T, error) {
	var zero T
	if raw == nil {
		return zero, nil
	}
	b, err := yaml.Marshal(raw)
	if err != nil {
		return zero, err
	}
	var out T
	if err := yaml.Unmarshal(b, &out); err != nil {
		return zero, err
	}
	return out, nil
}

type PlayerComponentSpec struct {
	MoveSpeed            float64 `yaml:"move_speed"`
	JumpSpeed            float64 `yaml:"jump_speed"`
	JumpHoldFrames       int     `yaml:"jump_hold_frames"`
	JumpHoldBoost        float64 `yaml:"jump_hold_boost"`
	CoyoteFrames         int     `yaml:"coyote_frames"`
	WallGrabFrames       int     `yaml:"wall_grab_frames"`
	WallSlideSpeed       float64 `yaml:"wall_slide_speed"`
	WallJumpPush         float64 `yaml:"wall_jump_push"`
	WallJumpFrames       int     `yaml:"wall_jump_frames"`
	JumpBufferFrames     int     `yaml:"jump_buffer_frames"`
	AimSlowFactor        float64 `yaml:"aim_slow_factor"`
	HitFreezeFrames      int     `yaml:"hit_freeze_frames"`
	DamageShakeIntensity float64 `yaml:"damage_shake_intensity"`
}

type TransformComponentSpec struct {
	X        float64 `yaml:"x"`
	Y        float64 `yaml:"y"`
	ScaleX   float64 `yaml:"scale_x"`
	ScaleY   float64 `yaml:"scale_y"`
	Rotation float64 `yaml:"rotation"`
}

type SpriteComponentSpec struct {
	Image              string  `yaml:"image"`
	UseSource          bool    `yaml:"use_source"`
	OriginX            float64 `yaml:"origin_x"`
	OriginY            float64 `yaml:"origin_y"`
	CenterOriginIfZero bool    `yaml:"center_origin_if_zero"`
	FacingLeft         bool    `yaml:"facing_left"`
}

type RenderLayerComponentSpec struct {
	Index int `yaml:"index"`
}

type LineRenderComponentSpec struct {
	StartX    float64 `yaml:"start_x"`
	StartY    float64 `yaml:"start_y"`
	EndX      float64 `yaml:"end_x"`
	EndY      float64 `yaml:"end_y"`
	Width     float32 `yaml:"width"`
	Color     string  `yaml:"color"`
	AntiAlias bool    `yaml:"anti_alias"`
}

type CameraComponentSpec struct {
	TargetName string  `yaml:"target_name"`
	Zoom       float64 `yaml:"zoom"`
	Smoothness float64 `yaml:"smoothness"`
	LookOffset float64 `yaml:"look_offset"`
	LookSmooth float64 `yaml:"look_smooth"`
}

type AIComponentSpec struct {
	MoveSpeed    float64 `yaml:"move_speed"`
	FollowRange  float64 `yaml:"follow_range"`
	AttackRange  float64 `yaml:"attack_range"`
	AttackFrames int     `yaml:"attack_frames"`
}

type PathfindingComponentSpec struct {
	GridSize      float64 `yaml:"grid_size"`
	RepathFrames  int     `yaml:"repath_frames"`
	DebugNodeSize float64 `yaml:"debug_node_size"`
}

type AIFSMEmbeddedStateSpec struct {
	OnEnter []map[string]any `yaml:"on_enter"`
	While   []map[string]any `yaml:"while"`
	OnExit  []map[string]any `yaml:"on_exit"`
}

type AIFSMEmbeddedSpec struct {
	Initial     string                            `yaml:"initial"`
	States      map[string]AIFSMEmbeddedStateSpec `yaml:"states"`
	Transitions map[string][]map[string]any       `yaml:"transitions"`
}

type AIConfigComponentSpec struct {
	FSM    string             `yaml:"fsm"`
	Script string             `yaml:"script"`
	Spec   *AIFSMEmbeddedSpec `yaml:"spec"`
}

type AnimationDefComponentSpec struct {
	Row        int     `yaml:"row"`
	ColStart   int     `yaml:"col_start"`
	FrameCount int     `yaml:"frame_count"`
	FrameW     int     `yaml:"frame_w"`
	FrameH     int     `yaml:"frame_h"`
	FPS        float64 `yaml:"fps"`
	Loop       bool    `yaml:"loop"`
}

type AnimationComponentSpec struct {
	Sheet      string                               `yaml:"sheet"`
	Defs       map[string]AnimationDefComponentSpec `yaml:"defs"`
	Current    string                               `yaml:"current"`
	Frame      int                                  `yaml:"frame"`
	FrameTimer int                                  `yaml:"frame_timer"`
	Playing    bool                                 `yaml:"playing"`
}

type AudioClipSpec struct {
	Name   string  `yaml:"name"`
	File   string  `yaml:"file"`
	Volume float64 `yaml:"volume"`
}

type AudioComponentSpec struct {
	Clips    []AudioClipSpec `yaml:"clips"`
	Autoplay []string        `yaml:"autoplay"`
}

type PhysicsBodyComponentSpec struct {
	Width              float64 `yaml:"width"`
	Height             float64 `yaml:"height"`
	Radius             float64 `yaml:"radius"`
	Mass               float64 `yaml:"mass"`
	Friction           float64 `yaml:"friction"`
	Elasticity         float64 `yaml:"elasticity"`
	Static             bool    `yaml:"static"`
	AlignTopLeft       bool    `yaml:"align_top_left"`
	OffsetX            float64 `yaml:"offset_x"`
	OffsetY            float64 `yaml:"offset_y"`
	ScaleWithTransform bool    `yaml:"scale_with_transform"`
	DefaultWidth       float64 `yaml:"default_width"`
	DefaultHeight      float64 `yaml:"default_height"`
}

type CollisionLayerComponentSpec struct {
	Category uint32 `yaml:"category"`
	Mask     uint32 `yaml:"mask"`
}

type RepulsionLayerComponentSpec struct {
	Category uint32 `yaml:"category"`
	Mask     uint32 `yaml:"mask"`
}

type GravityScaleComponentSpec struct {
	Scale float64 `yaml:"scale"`
}

type HazardComponentSpec struct {
	Width              float64 `yaml:"width"`
	Height             float64 `yaml:"height"`
	OffsetX            float64 `yaml:"offset_x"`
	OffsetY            float64 `yaml:"offset_y"`
	AutoSizeFromSprite bool    `yaml:"auto_size_from_sprite"`
	ScaleWithTransform bool    `yaml:"scale_with_transform"`
}

type HealthComponentSpec struct {
	Initial int `yaml:"initial"`
	Current int `yaml:"current"`
}

type HitboxComponentSpec struct {
	Width   float64 `yaml:"width"`
	Height  float64 `yaml:"height"`
	OffsetX float64 `yaml:"offset_x"`
	OffsetY float64 `yaml:"offset_y"`
	Damage  int     `yaml:"damage"`
	Anim    string  `yaml:"anim"`
	Frames  []int   `yaml:"frames"`
}

type HurtboxComponentSpec struct {
	Width   float64 `yaml:"width"`
	Height  float64 `yaml:"height"`
	OffsetX float64 `yaml:"offset_x"`
	OffsetY float64 `yaml:"offset_y"`
}

type AnchorComponentSpec struct {
	TargetX float64 `yaml:"target_x"`
	TargetY float64 `yaml:"target_y"`
	Speed   float64 `yaml:"speed"`
}

type AIPhaseComponentSpec struct {
	Name                string                      `yaml:"name"`
	StartWhen           []map[string]any            `yaml:"start_when"`
	TransitionOverrides map[string][]map[string]any `yaml:"transition_overrides"`
	OnEnter             []map[string]any            `yaml:"on_enter"`
}

type AIPhaseControllerComponentSpec struct {
	ResetStateOnPhaseChange *bool                  `yaml:"reset_state_on_phase_change"`
	Phases                  []AIPhaseComponentSpec `yaml:"phases"`
}

type ArenaNodeComponentSpec struct {
	Group             string `yaml:"group"`
	Active            *bool  `yaml:"active"`
	HazardEnabled     *bool  `yaml:"hazard_enabled"`
	TransitionEnabled *bool  `yaml:"transition_enabled"`
}

type RawState struct {
	OnEnter []map[string]any `yaml:"on_enter"`
	While   []map[string]any `yaml:"while"`
	OnExit  []map[string]any `yaml:"on_exit"`
}

type RawFSM struct {
	Initial     string              `yaml:"initial"`
	States      map[string]RawState `yaml:"states"`
	Transitions map[string]any      `yaml:"transitions"`
}

func LoadRawFSM(path string) (RawFSM, error) {
	return LoadSpec[RawFSM](path)
}

type PreviewSpec struct {
	Animation *AnimationSpec `yaml:"animation"`
	Sprite    *SpriteSpec    `yaml:"sprite"`
}

func LoadPreviewSpec(path string) (PreviewSpec, error) {
	return LoadSpec[PreviewSpec](path)
}
