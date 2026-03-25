package component

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type Particle struct {
	X, Y           float64
	AccelX, AccelY float64
	VelX, VelY     float64
	Life           int
}

func (p *Particle) Update() {
	p.VelX += p.AccelX
	p.VelY += p.AccelY
	p.X += p.VelX
	p.Y += p.VelY
	p.Life--
	p.AccelX = 0
	p.AccelY = 0
}

func (p *Particle) ApplyForce(accelX, accelY float64) {
	p.AccelX += accelX
	p.AccelY += accelY
}

func (p *Particle) IsDead() bool {
	return p.Life <= 0
}

type ParticleEmitter struct {
	Pool      sync.Pool
	Particles []*Particle

	TotalParticles int // Total number of particles to emit
	Lifetime       int // Lifetime of each particle in frames

	Burst      bool // Whether to emit all particles at once (burst) or one per frame
	Continuous bool // Whether to continuously emit particles
	HasGravity bool // Whether particles are affected by gravity

	Image *ebiten.Image // Image for the particles

	HasEmittedAtLeastOnce bool // Internal flag to track if at least one burst has occurred
}

var ParticleEmitterComponent = NewComponent[ParticleEmitter]()
