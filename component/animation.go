package component

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Animation provides a simple frame-based animator for a horizontal/rectangular
// spritesheet. Frames are laid out left-to-right, top-to-bottom.
type Animation struct {
	Sheet      *ebiten.Image
	FrameW     int
	FrameH     int
	FrameCount int
	Cols       int
	FPS        int
	Loop       bool

	current     int
	tick        int
	ticksPerFrm int
	startIndex  int
	frames      []*ebiten.Image
}

// NewAnimation creates an Animation. `sheet` is the full spritesheet image.
// `frameW`/`frameH` are the per-frame pixel size. `frameCount` is how many
// frames to read (use 0 to infer from sheet size). `fps` is frames per second
// for the animation (defaults to 12 if <= 0). `loop` controls whether animation
// should wrap.
func NewAnimation(sheet *ebiten.Image, frameW, frameH, frameCount, fps int, loop bool) *Animation {
	if sheet == nil || frameW <= 0 || frameH <= 0 {
		return &Animation{}
	}
	if fps <= 0 {
		fps = 12
	}
	bounds := sheet.Bounds()
	cols := bounds.Dx() / frameW
	rows := bounds.Dy() / frameH
	maxFrames := cols * rows
	if frameCount <= 0 || frameCount > maxFrames {
		frameCount = maxFrames
	}
	ticks := int(math.Max(1, math.Round(60.0/float64(fps))))
	a := &Animation{
		Sheet:       sheet,
		FrameW:      frameW,
		FrameH:      frameH,
		FrameCount:  frameCount,
		Cols:        cols,
		FPS:         fps,
		Loop:        loop,
		current:     0,
		tick:        0,
		ticksPerFrm: ticks,
		startIndex:  0,
	}
	a.buildFrames()
	return a
}

// NewAnimationRow creates an animation that starts at the given row (0-based)
// and reads `frameCount` frames left-to-right. If the requested frames exceed
// the row length they will continue onto subsequent rows.
func NewAnimationRow(sheet *ebiten.Image, frameW, frameH, row, frameCount, fps int, loop bool) *Animation {
	a := NewAnimation(sheet, frameW, frameH, frameCount, fps, loop)
	if a.Sheet == nil {
		return a
	}
	if row < 0 {
		row = 0
	}
	a.startIndex = row * (a.Sheet.Bounds().Dx() / a.FrameW)
	a.buildFrames()
	return a
}

// buildFrames slices the sheet into individual *ebiten.Image frames starting
// at a.startIndex and stores them in a.frames.
func (a *Animation) buildFrames() {
	if a == nil || a.Sheet == nil || a.FrameCount <= 0 {
		return
	}
	a.frames = make([]*ebiten.Image, a.FrameCount)
	for i := 0; i < a.FrameCount; i++ {
		idx := a.startIndex + i
		col := idx % a.Cols
		row := idx / a.Cols
		sx := col * a.FrameW
		sy := row * a.FrameH
		r := image.Rect(sx, sy, sx+a.FrameW, sy+a.FrameH)
		sub := a.Sheet.SubImage(r)
		a.frames[i] = ebiten.NewImageFromImage(sub)
	}
}

// Update advances the animation according to the configured FPS. Call once per
// game update (typically 60 times per second).
func (a *Animation) Update() {
	if a == nil || a.Sheet == nil || a.FrameCount <= 1 {
		return
	}
	a.tick++
	if a.tick >= a.ticksPerFrm {
		a.tick = 0
		a.current++
		if a.current >= a.FrameCount {
			if a.Loop {
				a.current = 0
			} else {
				a.current = a.FrameCount - 1
			}
		}
	}
}

// Reset sets the animation back to the first frame.
func (a *Animation) Reset() {
	if a == nil {
		return
	}
	a.current = 0
	a.tick = 0
}

// SetFrame jumps to a specific frame index.
func (a *Animation) SetFrame(i int) {
	if a == nil || a.FrameCount == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= a.FrameCount {
		i = a.FrameCount - 1
	}
	a.current = i
	a.tick = 0
}

// Draw draws the current frame at the given position. If `op` is nil a new
// DrawImageOptions will be used.
func (a *Animation) Draw(screen *ebiten.Image, op *ebiten.DrawImageOptions) {
	if a == nil || a.Sheet == nil || a.FrameCount == 0 {
		return
	}
	// compute absolute frame index including startIndex
	idx := a.startIndex + (a.current % a.FrameCount)
	if idx < 0 {
		idx = 0
	}
	col := idx % a.Cols
	row := idx / a.Cols
	sx := col * a.FrameW
	sy := row * a.FrameH

	// draw from prebuilt frame images if available
	if a.frames != nil && len(a.frames) > 0 {
		fi := a.current % a.FrameCount
		if fi < 0 {
			fi = 0
		}
		frm := a.frames[fi]
		var dop ebiten.DrawImageOptions
		if op != nil {
			dop = *op
		}
		dop.Filter = ebiten.FilterNearest
		screen.DrawImage(frm, &dop)
		return
	}
	// fallback: draw whole sheet region (shouldn't happen when frames built)
	var dop ebiten.DrawImageOptions
	if op != nil {
		dop = *op
	}
	dop.Filter = ebiten.FilterNearest
	sub := a.Sheet.SubImage(image.Rect(sx, sy, sx+a.FrameW, sy+a.FrameH)).(*ebiten.Image)
	screen.DrawImage(sub, &dop)
}

// Size returns the frame width/height.
func (a *Animation) Size() (int, int) { return a.FrameW, a.FrameH }
