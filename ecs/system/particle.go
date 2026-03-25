package system

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

var (
	colorWhite = color.Color(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
)

type ParticleSystem struct{}

const particleRadius = 2.0

func NewParticleSystem() *ParticleSystem { return &ParticleSystem{} }

func (s *ParticleSystem) Update(w *ecs.World) {
	ecs.ForEach2(w, component.TransformComponent.Kind(), component.ParticleEmitterComponent.Kind(), func(e ecs.Entity, t *component.Transform, emitter *component.ParticleEmitter) {
		addParticle := func() {
			particle := emitter.Pool.Get().(*component.Particle)

			particle.X = t.X
			particle.Y = t.Y
			particle.VelX = rand.Float64()*2 - 1    // Random horizontal velocity between -1 and 1
			particle.VelY = -(rand.Float64()*2 + 1) // Random upward velocity between -1 and -3
			particle.Life = emitter.Lifetime

			emitter.Particles = append(emitter.Particles, particle)
		}

		if len(emitter.Particles) < cap(emitter.Particles) && (emitter.Continuous || !emitter.HasEmittedAtLeastOnce) {
			if emitter.Burst {
				for i := 0; i < cap(emitter.Particles)-len(emitter.Particles); i++ {
					addParticle()
				}
			} else {
				addParticle()
			}
		} else {
			emitter.HasEmittedAtLeastOnce = true
		}

		live := emitter.Particles[:0]
		for _, particle := range emitter.Particles {

			if emitter.HasGravity {
				particle.ApplyForce(0, 0.1) // Gravity
			}

			particle.Update()

			if particle.IsDead() {
				emitter.Pool.Put(particle)
			} else {
				live = append(live, particle)
			}
		}
		emitter.Particles = live
	})
}

func (s *ParticleSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}

	camX, camY := 0.0, 0.0
	zoom := 1.0
	if camEntity, ok := ecs.First(w, component.CameraComponent.Kind()); ok {
		if camTransform, ok := ecs.Get(w, camEntity, component.TransformComponent.Kind()); ok {
			camX, camY, _, _, _ = resolvedTransform(camTransform)
		}
		if camComp, ok := ecs.Get(w, camEntity, component.CameraComponent.Kind()); ok && camComp.Zoom > 0 {
			zoom = camComp.Zoom
		}
	}

	target, ok := worldRenderTarget(screen, activeLevelBounds(w), camX, camY, zoom)
	if !ok {
		return
	}

	radius := particleRadius * zoom
	if radius < 1 {
		radius = 1
	}

	particleColor := colorWhite
	ecs.ForEach(w, component.ParticleEmitterComponent.Kind(), func(e ecs.Entity, emitter *component.ParticleEmitter) {
		c, hasColor := ecs.Get(w, e, component.ColorComponent.Kind())
		if hasColor {
			particleColor = componentColorToColorColor(*c)
		}

		for _, particle := range emitter.Particles {
			if particle.IsDead() {
				continue
			}

			sx := (particle.X - camX) * zoom
			sy := (particle.Y - camY) * zoom

			if emitter.Image != nil {
				op := &ebiten.DrawImageOptions{}

				if hasColor {
					op.ColorScale.ScaleWithColor(particleColor)
				}

				op.GeoM.Scale(emitter.Scale.X*zoom, emitter.Scale.Y*zoom)
				op.GeoM.Translate(sx-float64(emitter.Image.Bounds().Dx())/2, sy-float64(emitter.Image.Bounds().Dy())/2)
				target.DrawImage(emitter.Image, op)
			} else {
				vector.DrawFilledCircle(target, float32(sx), float32(sy), float32(radius), particleColor, true)
			}
		}
	})
}

func componentColorToColorColor(c component.Color) color.Color {
	r := uint8(c.R * 255)
	g := uint8(c.G * 255)
	b := uint8(c.B * 255)
	a := uint8(c.A * 255)
	return color.NRGBA{R: r, G: g, B: b, A: a}
}
