package systems

import (
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/component"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/components"
	"github.com/milk9111/sidescroller/ecs/render"
)

// RenderSystem draws sprites and animations.
type RenderSystem struct{}

// NewRenderSystem creates a RenderSystem.
func NewRenderSystem() *RenderSystem {
	return &RenderSystem{}
}

// Update is a no-op (render occurs in Draw).
func (s *RenderSystem) Update(w *ecs.World) {}

// Draw renders all entities with Transform + Sprite, sorted by layer.
func (s *RenderSystem) Draw(w *ecs.World, screen *ebiten.Image, camX, camY, zoom float64) {
	if w == nil || screen == nil {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}
	sp := w.Sprites()
	tr := w.Transforms()
	if sp == nil || tr == nil {
		return
	}

	type item struct {
		id    int
		layer int
		y     float32
	}

	items := make([]item, 0, len(sp.Entities()))
	for _, id := range sp.Entities() {
		v := tr.Get(id)
		if v == nil {
			continue
		}
		tx, ok := v.(*components.Transform)
		if !ok || tx == nil {
			continue
		}
		spv := sp.Get(id)
		spr, ok := spv.(*components.Sprite)
		if !ok || spr == nil {
			continue
		}
		items = append(items, item{id: id, layer: spr.Layer, y: tx.Y})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].layer == items[j].layer {
			return items[i].y < items[j].y
		}
		return items[i].layer < items[j].layer
	})

	for _, it := range items {
		id := it.id
		v := tr.Get(id)
		tx, ok := v.(*components.Transform)
		if !ok || tx == nil {
			continue
		}
		spv := sp.Get(id)
		spr, ok := spv.(*components.Sprite)
		if !ok || spr == nil {
			continue
		}

		if anim := w.GetAnimator(ecs.Entity{ID: id, Gen: 0}); anim != nil && anim.Anim != nil {
			drawAnimation(screen, anim.Anim, spr, tx, camX, camY, zoom)
			continue
		}
		drawSprite(screen, spr, tx, camX, camY, zoom)
	}
}

func drawSprite(screen *ebiten.Image, spr *components.Sprite, tx *components.Transform, camX, camY, zoom float64) {
	if screen == nil || spr == nil || tx == nil {
		return
	}
	img := render.GetImage(spr.ImageKey)
	if img == nil && spr.ImageKey != "" {
		if im, err := render.LoadImage(spr.ImageKey); err == nil {
			img = im
		}
	}
	if img == nil {
		return
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return
	}

	scaleX := 1.0
	scaleY := 1.0
	if spr.Width > 0 {
		scaleX = float64(spr.Width) / float64(w)
	}
	if spr.Height > 0 {
		scaleY = float64(spr.Height) / float64(h)
	}

	fx := 1.0
	fy := 1.0
	if spr.FlipX {
		fx = -1.0
	}
	if spr.FlipY {
		fy = -1.0
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX*zoom*fx, scaleY*zoom*fy)
	if spr.FlipX {
		op.GeoM.Translate(float64(w)*scaleX*zoom, 0)
	}
	if spr.FlipY {
		op.GeoM.Translate(0, float64(h)*scaleY*zoom)
	}

	txScreen := (float64(tx.X) + float64(spr.OffsetX) - camX) * zoom
	tyScreen := (float64(tx.Y) + float64(spr.OffsetY) - camY) * zoom
	op.GeoM.Translate(math.Round(txScreen), math.Round(tyScreen))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(img, op)
}

func drawAnimation(screen *ebiten.Image, anim *component.Animation, spr *components.Sprite, tx *components.Transform, camX, camY, zoom float64) {
	if screen == nil || anim == nil || spr == nil || tx == nil {
		return
	}
	w := anim.FrameW
	h := anim.FrameH
	if w <= 0 || h <= 0 {
		return
	}

	scaleX := 1.0
	scaleY := 1.0
	if spr.Width > 0 {
		scaleX = float64(spr.Width) / float64(w)
	}
	if spr.Height > 0 {
		scaleY = float64(spr.Height) / float64(h)
	}

	fx := 1.0
	fy := 1.0
	if spr.FlipX {
		fx = -1.0
	}
	if spr.FlipY {
		fy = -1.0
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX*zoom*fx, scaleY*zoom*fy)
	if spr.FlipX {
		op.GeoM.Translate(float64(w)*scaleX*zoom, 0)
	}
	if spr.FlipY {
		op.GeoM.Translate(0, float64(h)*scaleY*zoom)
	}

	txScreen := (float64(tx.X) + float64(spr.OffsetX) - camX) * zoom
	tyScreen := (float64(tx.Y) + float64(spr.OffsetY) - camY) * zoom
	op.GeoM.Translate(math.Round(txScreen), math.Round(tyScreen))
	op.Filter = ebiten.FilterNearest
	anim.Draw(screen, op)
}
