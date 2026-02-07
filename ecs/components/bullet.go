package components

import "github.com/milk9111/sidescroller/component"

// Bullet stores projectile data.
type Bullet struct {
	OwnerID    int
	Damage     component.Damage
	Width      float32
	Height     float32
	Rotation   float64
	AgeFrames  int
	LifeFrames int
}
