package component

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type AnimationDef struct {
	Name       string
	Row        int
	ColStart   int // start column (frame 0)
	FrameCount int
	FrameW     int
	FrameH     int
	FPS        float64
	Loop       bool
}

type Animation struct {
	Sheet      *ebiten.Image
	Defs       map[string]AnimationDef
	Current    string
	Frame      int
	FrameTimer int
	Playing    bool
}

var AnimationComponent = NewComponent[Animation]()
