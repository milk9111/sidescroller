package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/prefabs"
)

const (
	previewParticleRadius = 4.0
	previewZoom           = 3.0
	previewSeed           = 1337
	previewGravity        = 0.1
)

type previewParticle struct {
	X, Y           float64
	VelX, VelY     float64
	AccelX, AccelY float64
	Life           int
}

func (p *previewParticle) update() {
	p.VelX += p.AccelX
	p.VelY += p.AccelY
	p.X += p.VelX
	p.Y += p.VelY
	p.Life--
	p.AccelX = 0
	p.AccelY = 0
}

func (p *previewParticle) applyForce(accelX, accelY float64) {
	p.AccelX += accelX
	p.AccelY += accelY
}

func (p *previewParticle) isDead() bool {
	return p.Life <= 0
}

type particlePreview struct {
	spec                  prefabs.ParticleEmitterComponentSpec
	particles             []previewParticle
	hasEmittedAtLeastOnce bool
	playing               bool
	rng                   *rand.Rand
	image                 *ebiten.Image
	imageCache            map[string]*ebiten.Image
	tint                  color.NRGBA
	emittedTotal          int
}

func newParticlePreview() *particlePreview {
	p := &particlePreview{
		playing:    true,
		imageCache: make(map[string]*ebiten.Image),
		tint:       color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	}
	p.reset()
	return p
}

func (p *particlePreview) reset() {
	p.particles = p.particles[:0]
	p.hasEmittedAtLeastOnce = false
	p.rng = rand.New(rand.NewSource(previewSeed))
	p.emittedTotal = 0
}

func (p *particlePreview) SetSpec(spec prefabs.ParticleEmitterComponentSpec) error {
	img, err := p.resolveImage(spec.Image)
	if err != nil {
		// tolerate missing/failed image load; fallback to vector rendering
		img = nil
	}
	tint, err := parseNRGBA(spec.Color)
	if err != nil {
		return err
	}
	p.spec = spec
	p.image = img
	p.tint = tint
	p.Restart()
	return nil
}

func (p *particlePreview) resolveImage(path string) (*ebiten.Image, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, nil
	}
	if img, ok := p.imageCache[trimmed]; ok {
		return img, nil
	}
	img, err := assets.LoadImage(trimmed)
	if err != nil {
		return nil, fmt.Errorf("load particle image %q: %w", trimmed, err)
	}
	p.imageCache[trimmed] = img
	return img, nil
}

func (p *particlePreview) Play() {
	p.playing = true
}

func (p *particlePreview) Pause() {
	p.playing = false
}

func (p *particlePreview) Stop() {
	p.playing = false
	p.reset()
}

func (p *particlePreview) Restart() {
	p.reset()
	p.playing = true
}

func (p *particlePreview) Update(originX, originY float64) {
	if p == nil || !p.playing {
		return
	}
	addParticle := func() {
		if p.spec.TotalParticles <= 0 {
			return
		}
		if len(p.particles) >= p.spec.TotalParticles {
			return
		}
		particle := previewParticle{
			X:    originX,
			Y:    originY,
			VelX: p.rng.Float64()*2 - 1,
			VelY: -(p.rng.Float64()*2 + 1),
			Life: p.spec.Lifetime,
		}
		p.particles = append(p.particles, particle)
		p.emittedTotal++
	}

	// Spawn logic:
	// - Burst: emit all particles once (on first update after spec set/restart)
	// - Continuous: emit particles gradually each update until reaching TotalParticles
	// - Neither: do not emit automatically
	if len(p.particles) < p.spec.TotalParticles {
		if p.spec.Burst {
			if !p.hasEmittedAtLeastOnce {
				remaining := p.spec.TotalParticles - len(p.particles)
				for i := 0; i < remaining; i++ {
					addParticle()
				}
				p.hasEmittedAtLeastOnce = true
				p.emittedTotal = p.spec.TotalParticles
			}
		} else if p.spec.Continuous {
			addParticle()
		} else {
			// Neither burst nor continuous: emit a stream until we've emitted the configured total,
			// then stop emitting even if particles die.
			if p.emittedTotal < p.spec.TotalParticles {
				addParticle()
			}
		}
	}

	live := p.particles[:0]
	for i := range p.particles {
		particle := p.particles[i]
		if p.spec.HasGravity {
			particle.applyForce(0, previewGravity)
		}
		particle.update()
		if !particle.isDead() {
			live = append(live, particle)
		}
	}
	p.particles = live
}

func (p *particlePreview) Draw(screen *ebiten.Image, rect image.Rectangle, originX, originY float64) {
	if p == nil || screen == nil || rect.Dx() <= 0 || rect.Dy() <= 0 {
		return
	}
	// treat zero scale as 1.0 to remain compatible with existing prefabs
	sxScale := p.spec.Scale.X
	syScale := p.spec.Scale.Y
	if sxScale == 0 {
		sxScale = 1.0
	}
	if syScale == 0 {
		syScale = 1.0
	}
	for i := range p.particles {
		particle := p.particles[i]
		if particle.isDead() {
			continue
		}
		sx := originX + (particle.X-originX)*previewZoom*sxScale
		sy := originY + (particle.Y-originY)*previewZoom*syScale
		if sx < float64(rect.Min.X)-64 || sx > float64(rect.Max.X)+64 || sy < float64(rect.Min.Y)-64 || sy > float64(rect.Max.Y)+64 {
			continue
		}

		if p.image != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(sxScale, syScale)
			op.GeoM.Translate(sx-float64(p.image.Bounds().Dx())*previewZoom*sxScale/2, sy-float64(p.image.Bounds().Dy())*previewZoom*syScale/2)
			op.ColorScale.ScaleWithColor(p.tint)
			screen.DrawImage(p.image, op)
			continue
		}

		// scale particle radius by average of X/Y scales and the preview image scale
		avgScale := (sxScale + syScale) / 2.0
		vector.DrawFilledCircle(screen, float32(sx), float32(sy), float32(previewParticleRadius*previewZoom*avgScale), p.tint, true)
	}
}
