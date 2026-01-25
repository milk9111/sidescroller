package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

type demoGame struct {
	frames      []*ebiten.Image
	frameCount  int
	current     int
	tick        int
	ticksPerFrm int
}

func (g *demoGame) Update() error {
	if g.frameCount <= 1 {
		return nil
	}
	g.tick++
	if g.tick >= g.ticksPerFrm {
		g.tick = 0
		g.current++
		if g.current >= g.frameCount {
			g.current = 0
		}
	}
	return nil
}

func (g *demoGame) Draw(screen *ebiten.Image) {
	// clear
	screen.Fill(color.RGBA{0x00, 0x00, 0x00, 0xff})
	if g.frameCount == 0 {
		return
	}
	fw := g.frames[0].Bounds().Dx()
	fh := g.frames[0].Bounds().Dy()
	sx := (512 - fw) / 2
	sy := (512 - fh) / 2
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(sx), float64(sy))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(g.frames[g.current], op)
}

func (g *demoGame) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 512, 512
}

func loadFrames(path string, frameW, frameH, count, fps int) ([]*ebiten.Image, int) {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Printf("failed to read %s: %v", path, err)
		return nil, 0
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Printf("failed to decode %s: %v", path, err)
		return nil, 0
	}
	sheet := ebiten.NewImageFromImage(img)
	cols := sheet.Bounds().Dx() / frameW
	rows := sheet.Bounds().Dy() / frameH
	maxFrames := cols * rows
	if count <= 0 || count > maxFrames {
		count = maxFrames
	}
	frames := make([]*ebiten.Image, count)
	for i := 0; i < count; i++ {
		col := i % cols
		row := i / cols
		r := image.Rect(col*frameW, row*frameH, col*frameW+frameW, row*frameH+frameH)
		sub := sheet.SubImage(r).(image.Image)
		frames[i] = ebiten.NewImageFromImage(sub)
	}
	ticks := 1
	if fps > 0 {
		ticks = int(60 / fps)
		if ticks < 1 {
			ticks = 1
		}
	}
	return frames, ticks
}

func main() {
	frames, ticks := loadFrames("assets/player-Sheet.png", 128, 128, 9, 12)
	g := &demoGame{frames: frames, frameCount: len(frames), current: 0, tick: 0, ticksPerFrm: ticks}
	ebiten.SetWindowSize(512, 512)
	ebiten.SetWindowTitle("Player Idle Demo")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
