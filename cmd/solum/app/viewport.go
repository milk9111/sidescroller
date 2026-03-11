package app

import (
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"math"
	"strings"

	g "github.com/AllenDang/giu"
	"github.com/milk9111/sidescroller/levels"
)

func (a *App) buildViewportWidget() g.Widget {
	state := a.state
	return g.Custom(func() {
		pos := g.GetCursorScreenPos()
		availW, availH := g.GetAvailableRegion()
		width := int(math.Max(240, float64(availW)))
		height := int(math.Max(320, float64(availH)))
		rect := image.Rect(pos.X, pos.Y, pos.X+width, pos.Y+height)

		g.InvisibleButton().ID("solum-viewport").Size(float32(width), float32(height)).Build()
		mouse := g.GetMousePos()
		input := ViewportInput{
			MouseX:             mouse.X,
			MouseY:             mouse.Y,
			WheelY:             mouseWheelY(),
			LeftDown:           g.IsMouseDown(g.MouseButtonLeft),
			RightDown:          g.IsMouseDown(g.MouseButtonRight),
			MiddleDown:         g.IsMouseDown(g.MouseButtonMiddle),
			LeftJustPressed:    g.IsMouseClicked(g.MouseButtonLeft),
			LeftJustReleased:   g.IsMouseReleased(g.MouseButtonLeft),
			RightJustPressed:   g.IsMouseClicked(g.MouseButtonRight),
			RightJustReleased:  g.IsMouseReleased(g.MouseButtonRight),
			MiddleJustPressed:  g.IsMouseClicked(g.MouseButtonMiddle),
			MiddleJustReleased: g.IsMouseReleased(g.MouseButtonMiddle),
			Hovered:            g.IsItemHovered(),
		}
		if state.Overview.Open {
			state.UpdateOverview(rect, input)
		} else {
			state.UpdateViewport(rect, input)
		}

		canvas := g.GetCanvas()
		state.drawViewport(canvas, rect)
	})
}

func (s *State) drawViewport(canvas *g.Canvas, rect image.Rectangle) {
	if s == nil || canvas == nil {
		return
	}
	if s.Overview.Open {
		s.drawOverview(canvas, rect)
		return
	}
	bg := color.RGBA{R: 18, G: 22, B: 32, A: 255}
	canvas.AddRectFilled(rect.Min, rect.Max, bg, 0, 0)
	canvas.AddRect(rect.Min, rect.Max, color.RGBA{R: 88, G: 98, B: 118, A: 255}, 0, 0, 1)

	g.PushClipRect(rect.Min, rect.Max, true)
	defer g.PopClipRect()

	levelMin, levelMax := s.levelScreenRect(rect)
	canvas.AddRectFilled(levelMin, levelMax, color.RGBA{R: 26, G: 32, B: 44, A: 255}, 0, 0)

	startX := maxInt(0, int(math.Floor(s.Camera.X/float64(tileSize))))
	startY := maxInt(0, int(math.Floor(s.Camera.Y/float64(tileSize))))
	endX := minInt(s.Document.Width, int(math.Ceil((s.Camera.X+s.Camera.CanvasW/s.Camera.Zoom)/float64(tileSize)))+1)
	endY := minInt(s.Document.Height, int(math.Ceil((s.Camera.Y+s.Camera.CanvasH/s.Camera.Zoom)/float64(tileSize)))+1)

	for layerIndex := range s.Document.Layers {
		layer := &s.Document.Layers[layerIndex]
		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				usage := layer.TilesetUsage[s.cellIndex(x, y)]
				if usage == nil || strings.TrimSpace(usage.Path) == "" {
					continue
				}
				cellMin, cellMax := s.cellScreenRect(rect, x, y)
				clr := tileColor(usage, layerIndex == s.CurrentLayer)
				canvas.AddRectFilled(cellMin, cellMax, clr, 0, 0)
				if s.Camera.Zoom >= 0.75 {
					outline := color.RGBA{R: 12, G: 16, B: 24, A: 180}
					canvas.AddRect(cellMin, cellMax, outline, 0, 0, 1)
				}
				if s.Camera.Zoom >= 1.6 {
					label := fmt.Sprintf("%d", usage.Index)
					canvas.AddText(cellMin.Add(image.Pt(4, 4)), color.RGBA{R: 245, G: 245, B: 245, A: 255}, label)
				}
			}
		}
	}

	for index, entity := range s.Document.Entities {
		if !entitySelectableOnCurrentLayer(entity, s.CurrentLayer) {
			continue
		}
		entityMin, entityMax := s.entityScreenRect(rect, entity)
		outline := color.RGBA{R: 80, G: 190, B: 255, A: 220}
		if index == s.SelectedEntity {
			outline = color.RGBA{R: 255, G: 214, B: 87, A: 255}
		}
		canvas.AddRect(entityMin, entityMax, outline, 0, 0, 2)
		if s.Camera.Zoom >= 1.4 {
			canvas.AddText(entityMin.Add(image.Pt(2, 2)), color.RGBA{R: 235, G: 240, B: 245, A: 255}, strings.TrimSpace(entity.Type))
		}
	}

	if s.Camera.Zoom >= 0.6 {
		grid := color.RGBA{R: 58, G: 66, B: 82, A: 150}
		for x := startX; x <= endX; x++ {
			sx := int(s.Camera.CanvasX + (float64(x*tileSize)-s.Camera.X)*s.Camera.Zoom)
			canvas.AddLine(image.Pt(sx, rect.Min.Y), image.Pt(sx, rect.Max.Y), grid, 1)
		}
		for y := startY; y <= endY; y++ {
			sy := int(s.Camera.CanvasY + (float64(y*tileSize)-s.Camera.Y)*s.Camera.Zoom)
			canvas.AddLine(image.Pt(rect.Min.X, sy), image.Pt(rect.Max.X, sy), grid, 1)
		}
	}

	for _, cell := range s.ToolStroke.Preview {
		if !s.withinLevel(cell.X, cell.Y) {
			continue
		}
		cellMin, cellMax := s.cellScreenRect(rect, cell.X, cell.Y)
		canvas.AddRect(cellMin, cellMax, color.RGBA{R: 255, G: 255, B: 255, A: 220}, 0, 0, 2)
	}
	if s.Pointer.HasCell {
		cellMin, cellMax := s.cellScreenRect(rect, s.Pointer.CellX, s.Pointer.CellY)
		canvas.AddRect(cellMin, cellMax, color.RGBA{R: 255, G: 255, B: 255, A: 120}, 0, 0, 2)
	}

	canvas.AddText(rect.Min.Add(image.Pt(12, 12)), color.RGBA{R: 230, G: 236, B: 245, A: 255}, fmt.Sprintf("%s  Zoom %.2fx  Camera %.0f, %.0f", s.DisplayLevelName(), s.Camera.Zoom, s.Camera.X, s.Camera.Y))
	canvas.AddText(rect.Min.Add(image.Pt(12, 30)), color.RGBA{R: 170, G: 182, B: 198, A: 255}, fmt.Sprintf("Pointer %d,%d  Tile %s", s.Pointer.CellX, s.Pointer.CellY, s.SelectedTileLabel()))
	canvas.AddText(rect.Min.Add(image.Pt(12, 48)), color.RGBA{R: 170, G: 182, B: 198, A: 255}, "LMB paint, MMB pan, wheel zoom, RMB sample")
}

func (s *State) levelScreenRect(rect image.Rectangle) (image.Point, image.Point) {
	left := int(s.Camera.CanvasX + (-s.Camera.X * s.Camera.Zoom))
	top := int(s.Camera.CanvasY + (-s.Camera.Y * s.Camera.Zoom))
	right := int(s.Camera.CanvasX + (float64(s.Document.Width*tileSize)-s.Camera.X)*s.Camera.Zoom)
	bottom := int(s.Camera.CanvasY + (float64(s.Document.Height*tileSize)-s.Camera.Y)*s.Camera.Zoom)
	if left < rect.Min.X {
		left = rect.Min.X
	}
	if top < rect.Min.Y {
		top = rect.Min.Y
	}
	if right > rect.Max.X {
		right = rect.Max.X
	}
	if bottom > rect.Max.Y {
		bottom = rect.Max.Y
	}
	return image.Pt(left, top), image.Pt(right, bottom)
}

func (s *State) cellScreenRect(rect image.Rectangle, cellX, cellY int) (image.Point, image.Point) {
	x := int(float64(rect.Min.X) + (float64(cellX*tileSize)-s.Camera.X)*s.Camera.Zoom)
	y := int(float64(rect.Min.Y) + (float64(cellY*tileSize)-s.Camera.Y)*s.Camera.Zoom)
	size := int(math.Max(1, s.Camera.Zoom*float64(tileSize)))
	return image.Pt(x, y), image.Pt(x+size, y+size)
}

func (s *State) entityScreenRect(rect image.Rectangle, entity levels.Entity) (image.Point, image.Point) {
	x := int(float64(rect.Min.X) + (float64(entity.X)-s.Camera.X)*s.Camera.Zoom)
	y := int(float64(rect.Min.Y) + (float64(entity.Y)-s.Camera.Y)*s.Camera.Zoom)
	size := int(math.Max(8, s.Camera.Zoom*float64(tileSize)))
	return image.Pt(x, y), image.Pt(x+size, y+size)
}

func tileColor(usage *levels.TileInfo, active bool) color.RGBA {
	if usage == nil {
		return color.RGBA{}
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(usage.Path))
	value := hasher.Sum32() + uint32(usage.BaseIndex+usage.Index+1)
	r := uint8(70 + (value & 0x3f))
	g := uint8(80 + ((value >> 6) & 0x5f))
	b := uint8(90 + ((value >> 12) & 0x5f))
	a := uint8(180)
	if active {
		a = 230
	}
	if usage.Auto {
		g = uint8(minInt(255, int(g)+25))
	}
	return color.RGBA{R: r, G: g, B: b, A: a}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
