package components

import "github.com/milk9111/sidescroller/component"

// Transform stores position and rotation in world space.
type Transform struct {
	X, Y   float32
	Angle  float64
	ScaleX float32
	ScaleY float32
}

// Velocity stores linear velocity.
type Velocity struct {
	VX, VY float32
}

// Acceleration stores linear acceleration.
type Acceleration struct {
	AX, AY float32
}

// Gravity stores gravity settings.
type Gravity struct {
	Enabled bool
	Value   float32
}

// Sprite stores render data.
type Sprite struct {
	ImageKey string
	Width    float32
	Height   float32
	OffsetX  float32
	OffsetY  float32
	FlipX    bool
	FlipY    bool
	Layer    int
}

// AnimationRef stores animation state.
type AnimationRef struct {
	ClipKey    string
	Frame      int
	FPS        int
	Loop       bool
	Playing    bool
	StartFrame int
}

// Collider stores collision info.
type Collider struct {
	Width         float32
	Height        float32
	OffsetX       float32
	OffsetY       float32
	Sensor        bool
	Static        bool
	FixedRotation bool
	IsEnemy       bool
	IsPlayer      bool
}

// GroundSensor defines a small sensor below a collider to detect grounded state.
type GroundSensor struct {
	Width   float32
	Height  float32
	OffsetX float32
	OffsetY float32
}

// WallSide indicates which side a wall contact is on.
type WallSide int

const (
	WallNone WallSide = iota
	WallLeft
	WallRight
)

// CollisionState stores the current collision flags.
type CollisionState struct {
	Grounded    bool
	Wall        WallSide
	HitHazard   bool
	GroundGrace int

	PrevGrounded  bool
	PrevWall      WallSide
	PrevHitHazard bool
}

// Health stores hit points and invulnerability frames.
type Health struct {
	Current float32
	Max     float32
	IFrames int
	Dead    bool

	OnDamage      func(h *Health, evt component.CombatEvent)
	OnDeath       func(h *Health, evt component.CombatEvent)
	OnIFrameStart func(h *Health)
	OnIFrameEnd   func(h *Health)
}

// Damage stores damage data for hitboxes.
type Damage struct {
	Amount         float32
	KnockbackX     float32
	KnockbackY     float32
	HitstunFrames  int
	CooldownFrames int
	IFrameFrames   int
	Faction        int
	MultiHit       bool
}

// Lifetime stores age and max age.
type Lifetime struct {
	AgeFrames int
	MaxFrames int
}
