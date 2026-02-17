package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/levels"
)

type levelOverviewTransition struct {
	To  string
	Dir string
}

type levelOverviewLevel struct {
	Name        string
	WidthTiles  int
	HeightTiles int
	WidthUnits  int
	HeightUnits int
	XUnits      int
	YUnits      int
	Placed      bool

	Issues      []string
	Transitions []levelOverviewTransition
}

type LevelOverviewView struct {
	active bool
	dirty  bool

	zoom      float64
	panX      float64
	panY      float64
	isPanning bool
	lastPanX  int
	lastPanY  int

	unitPixels float64
	gapUnits   int

	leftPanelWidth  int
	rightPanelWidth int

	levels       []levelOverviewLevel
	levelIndex   map[string]int
	hovered      int
	pressedNode  int
	pressScreenX int
	pressScreenY int

	lastError string
	framed    bool

	onLoadLevel   func(levelName string) error
	getCurrentLvl func() string
}

func NewLevelOverviewView(leftPanelWidth, rightPanelWidth int, onLoadLevel func(levelName string) error, getCurrentLvl func() string) *LevelOverviewView {
	return &LevelOverviewView{
		dirty:           true,
		zoom:            1.0,
		unitPixels:      16,
		gapUnits:        2,
		leftPanelWidth:  leftPanelWidth,
		rightPanelWidth: rightPanelWidth,
		onLoadLevel:     onLoadLevel,
		getCurrentLvl:   getCurrentLvl,
		pressedNode:     -1,
		hovered:         -1,
	}
}

func (v *LevelOverviewView) IsActive() bool {
	return v != nil && v.active
}

func (v *LevelOverviewView) Toggle() {
	if v == nil {
		return
	}
	v.active = !v.active
	if v.active {
		v.dirty = true
	}
}

func (v *LevelOverviewView) SetDirty() {
	if v == nil {
		return
	}
	v.dirty = true
}

func parseTransitionDirection(props map[string]interface{}) string {
	rawDir := ""
	if props == nil {
		return ""
	}
	if raw, ok := props["linked_id"]; ok {
		if s, ok := raw.(string); ok {
			rawDir = strings.ToLower(strings.TrimSpace(s))
		}
	}
	if rawDir == "" {
		if raw, ok := props["enter_dir"]; ok {
			if s, ok := raw.(string); ok {
				rawDir = strings.ToLower(strings.TrimSpace(s))
			}
		}
	}
	if rawDir == "" {
		return ""
	}
	// Transition direction fields describe how the player enters the destination
	// level, so the destination's world position is the opposite side relative to
	// the source level.
	if out := oppositeDir(rawDir); out != "" {
		return out
	}
	return rawDir
}

func dirDelta(dir string) (dx, dy int, ok bool) {
	switch strings.ToLower(strings.TrimSpace(dir)) {
	case "left":
		return -1, 0, true
	case "right":
		return 1, 0, true
	case "up", "top":
		return 0, -1, true
	case "down", "bottom":
		return 0, 1, true
	case "upper_left", "up_left", "top_left":
		return -1, -1, true
	case "upper_right", "up_right", "top_right":
		return 1, -1, true
	case "lower_left", "down_left", "bottom_left":
		return -1, 1, true
	case "lower_right", "down_right", "bottom_right":
		return 1, 1, true
	default:
		return 0, 0, false
	}
}

func oppositeDir(dir string) string {
	dx, dy, ok := dirDelta(dir)
	if !ok {
		return ""
	}
	switch {
	case dx == -1 && dy == 0:
		return "right"
	case dx == 1 && dy == 0:
		return "left"
	case dx == 0 && dy == -1:
		return "down"
	case dx == 0 && dy == 1:
		return "up"
	case dx == -1 && dy == -1:
		return "lower_right"
	case dx == 1 && dy == -1:
		return "lower_left"
	case dx == -1 && dy == 1:
		return "upper_right"
	case dx == 1 && dy == 1:
		return "upper_left"
	default:
		return ""
	}
}

func sizeUnits(tiles int) int {
	if tiles <= 0 {
		return 1
	}
	return int(math.Ceil(float64(tiles) / 5.0))
}

func (v *LevelOverviewView) rebuild() {
	v.lastError = ""
	entries, err := os.ReadDir("levels")
	if err != nil {
		v.lastError = fmt.Sprintf("failed to read levels/: %v", err)
		v.levels = nil
		v.levelIndex = nil
		v.dirty = false
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := normalizeLevelFileName(entry.Name())
		if strings.HasSuffix(strings.ToLower(name), ".json") {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	levelsData := make([]levelOverviewLevel, 0, len(names))
	levelIndex := make(map[string]int, len(names))
	issueSet := make(map[string]map[string]bool, len(names))

	for _, name := range names {
		path := filepath.Join("levels", name)
		bytes, readErr := os.ReadFile(path)
		if readErr != nil {
			continue
		}
		var lvl levels.Level
		if err := json.Unmarshal(bytes, &lvl); err != nil {
			continue
		}
		li := levelOverviewLevel{
			Name:        name,
			WidthTiles:  lvl.Width,
			HeightTiles: lvl.Height,
			WidthUnits:  sizeUnits(lvl.Width),
			HeightUnits: sizeUnits(lvl.Height),
		}
		for _, ent := range lvl.Entities {
			if !strings.EqualFold(ent.Type, "transition") || ent.Props == nil {
				continue
			}
			rawTarget, ok := ent.Props["to_level"]
			if !ok {
				continue
			}
			target, ok := rawTarget.(string)
			if !ok {
				continue
			}
			target = normalizeLevelFileName(target)
			dir := parseTransitionDirection(ent.Props)
			li.Transitions = append(li.Transitions, levelOverviewTransition{To: target, Dir: dir})
		}
		levelIndex[li.Name] = len(levelsData)
		levelsData = append(levelsData, li)
		issueSet[li.Name] = map[string]bool{}
	}

	addIssue := func(levelName, msg string) {
		if levelName == "" || msg == "" {
			return
		}
		m, ok := issueSet[levelName]
		if !ok {
			m = map[string]bool{}
			issueSet[levelName] = m
		}
		m[msg] = true
	}

	hasMultiplePairTransitions := func(a, b string) bool {
		if a == "" || b == "" || a == b {
			return false
		}
		count := 0
		for i := range levelsData {
			src := levelsData[i]
			if src.Name != a && src.Name != b {
				continue
			}
			for _, tr := range src.Transitions {
				if (src.Name == a && tr.To == b) || (src.Name == b && tr.To == a) {
					count++
				}
			}
		}
		return count > 1
	}

	expectedPos := func(src levelOverviewLevel, target levelOverviewLevel, dir string) (int, int, bool) {
		dx, dy, ok := dirDelta(dir)
		if !ok {
			return 0, 0, false
		}
		tx := src.XUnits
		ty := src.YUnits
		switch {
		case dx == -1:
			tx = src.XUnits - v.gapUnits - target.WidthUnits
		case dx == 1:
			tx = src.XUnits + src.WidthUnits + v.gapUnits
		case dx == 0:
			tx = src.XUnits
		}
		switch {
		case dy == -1:
			ty = src.YUnits - v.gapUnits - target.HeightUnits
		case dy == 1:
			ty = src.YUnits + src.HeightUnits + v.gapUnits
		case dy == 0:
			ty = src.YUnits
		}
		return tx, ty, true
	}

	if len(levelsData) > 0 {
		levelsData[0].XUnits = 0
		levelsData[0].YUnits = 0
		levelsData[0].Placed = true
	}

	for iter := 0; iter < len(levelsData)*4; iter++ {
		progress := false
		for i := range levelsData {
			src := &levelsData[i]
			for _, tr := range src.Transitions {
				targetIdx, exists := levelIndex[tr.To]
				if !exists {
					addIssue(src.Name, fmt.Sprintf("transition to missing level %s", tr.To))
					continue
				}
				if tr.To == src.Name {
					addIssue(src.Name, "self-transition")
					continue
				}
				dir := strings.ToLower(strings.TrimSpace(tr.Dir))
				if _, _, ok := dirDelta(dir); !ok {
					addIssue(src.Name, fmt.Sprintf("unknown transition direction '%s' to %s", tr.Dir, tr.To))
					continue
				}

				target := &levelsData[targetIdx]
				if src.Placed && !target.Placed {
					tx, ty, ok := expectedPos(*src, *target, dir)
					if ok {
						target.XUnits = tx
						target.YUnits = ty
						target.Placed = true
						progress = true
					}
				} else if !src.Placed && target.Placed {
					op := oppositeDir(dir)
					sx, sy, ok := expectedPos(*target, *src, op)
					if ok {
						src.XUnits = sx
						src.YUnits = sy
						src.Placed = true
						progress = true
					}
				} else if src.Placed && target.Placed {
					tx, ty, ok := expectedPos(*src, *target, dir)
					if ok && (target.XUnits != tx || target.YUnits != ty) {
						if hasMultiplePairTransitions(src.Name, target.Name) {
							continue
						}
						addIssue(src.Name, fmt.Sprintf("transition to %s (%s) conflicts with placement", tr.To, dir))
						addIssue(target.Name, fmt.Sprintf("placement conflict from %s (%s)", src.Name, dir))
					}
				}
			}
		}
		if !progress {
			break
		}
	}

	fallbackX := 0
	fallbackY := 0
	for i := range levelsData {
		if levelsData[i].Placed {
			continue
		}
		levelsData[i].XUnits = fallbackX
		levelsData[i].YUnits = fallbackY
		levelsData[i].Placed = true
		fallbackX += levelsData[i].WidthUnits + v.gapUnits + 4
		if fallbackX > 120 {
			fallbackX = 0
			fallbackY += 20
		}
		addIssue(levelsData[i].Name, "unanchored placement (no resolvable directional links)")
	}

	for i := range levelsData {
		dirTargets := map[string]map[string]bool{}
		for _, tr := range levelsData[i].Transitions {
			dir := strings.ToLower(strings.TrimSpace(tr.Dir))
			if _, _, ok := dirDelta(dir); !ok {
				continue
			}
			if dirTargets[dir] == nil {
				dirTargets[dir] = map[string]bool{}
			}
			dirTargets[dir][tr.To] = true
		}
		for dir, targets := range dirTargets {
			if len(targets) > 1 {
				addIssue(levelsData[i].Name, fmt.Sprintf("multiple levels use direction %s", dir))
			}
		}
	}

	type incomingRef struct {
		source string
		target string
		slot   string
	}
	incoming := map[string][]incomingRef{}
	for i := range levelsData {
		src := levelsData[i]
		for _, tr := range src.Transitions {
			if _, exists := levelIndex[tr.To]; !exists {
				continue
			}
			dir := strings.ToLower(strings.TrimSpace(tr.Dir))
			op := oppositeDir(dir)
			if op == "" {
				continue
			}
			key := tr.To + "|" + op
			incoming[key] = append(incoming[key], incomingRef{source: src.Name, target: tr.To, slot: op})
		}
	}
	for _, refs := range incoming {
		if len(refs) <= 1 {
			continue
		}
		sources := map[string]bool{}
		for _, ref := range refs {
			sources[ref.source] = true
		}
		// Allow duplicates when they are only between the same two levels.
		if len(sources) == 1 {
			continue
		}
		target := refs[0].target
		slot := refs[0].slot
		addIssue(target, fmt.Sprintf("more than one level uses incoming %s transition", slot))
		for _, ref := range refs {
			addIssue(ref.source, fmt.Sprintf("shares %s incoming slot on %s", slot, target))
		}
	}

	overlaps := func(a, b levelOverviewLevel) bool {
		return a.XUnits < b.XUnits+b.WidthUnits &&
			a.XUnits+a.WidthUnits > b.XUnits &&
			a.YUnits < b.YUnits+b.HeightUnits &&
			a.YUnits+a.HeightUnits > b.YUnits
	}

	pairTransitionCount := map[string]int{}
	pairKey := func(a, b string) string {
		if a < b {
			return a + "|" + b
		}
		return b + "|" + a
	}
	for i := range levelsData {
		src := levelsData[i]
		for _, tr := range src.Transitions {
			if tr.To == "" || tr.To == src.Name {
				continue
			}
			if _, ok := levelIndex[tr.To]; !ok {
				continue
			}
			pairTransitionCount[pairKey(src.Name, tr.To)]++
		}
	}

	shouldAllowPairConflict := func(a, b string) bool {
		return pairTransitionCount[pairKey(a, b)] > 1
	}

	for i := 0; i < len(levelsData); i++ {
		for j := i + 1; j < len(levelsData); j++ {
			if overlaps(levelsData[i], levelsData[j]) {
				if shouldAllowPairConflict(levelsData[i].Name, levelsData[j].Name) {
					continue
				}
				addIssue(levelsData[i].Name, fmt.Sprintf("box collision with %s", levelsData[j].Name))
				addIssue(levelsData[j].Name, fmt.Sprintf("box collision with %s", levelsData[i].Name))
			}
		}
	}

	for i := range levelsData {
		issues := issueSet[levelsData[i].Name]
		if len(issues) == 0 {
			continue
		}
		levelsData[i].Issues = make([]string, 0, len(issues))
		for msg := range issues {
			levelsData[i].Issues = append(levelsData[i].Issues, msg)
		}
		sort.Strings(levelsData[i].Issues)
	}

	v.levels = levelsData
	v.levelIndex = levelIndex
	v.dirty = false
	v.framed = false
}

func (v *LevelOverviewView) canvasBounds(screenW, screenH int) (x, y, w, h int) {
	x = v.leftPanelWidth
	y = 0
	w = screenW - v.leftPanelWidth - v.rightPanelWidth
	h = screenH
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return
}

func (v *LevelOverviewView) worldScale() float64 {
	return v.unitPixels * v.zoom
}

func (v *LevelOverviewView) worldToScreen(canvasX int, wx, wy float64) (sx, sy float64) {
	s := v.worldScale()
	sx = float64(canvasX) + v.panX + wx*s
	sy = v.panY + wy*s
	return
}

func (v *LevelOverviewView) screenToWorld(canvasX int, sx, sy int) (wx, wy float64) {
	s := v.worldScale()
	wx = (float64(sx-canvasX) - v.panX) / s
	wy = (float64(sy) - v.panY) / s
	return
}

func (v *LevelOverviewView) levelAtScreenPos(screenX, screenY, canvasX int) int {
	for i := range v.levels {
		lvl := v.levels[i]
		sx, sy := v.worldToScreen(canvasX, float64(lvl.XUnits), float64(lvl.YUnits))
		sw := float64(lvl.WidthUnits) * v.worldScale()
		sh := float64(lvl.HeightUnits) * v.worldScale()
		if float64(screenX) >= sx && float64(screenX) <= sx+sw && float64(screenY) >= sy && float64(screenY) <= sy+sh {
			return i
		}
	}
	return -1
}

func (v *LevelOverviewView) frameToCanvas(canvasW, canvasH int) {
	if len(v.levels) == 0 {
		v.panX = 0
		v.panY = 0
		v.framed = true
		return
	}
	minX := v.levels[0].XUnits
	minY := v.levels[0].YUnits
	maxX := v.levels[0].XUnits + v.levels[0].WidthUnits
	maxY := v.levels[0].YUnits + v.levels[0].HeightUnits
	for i := 1; i < len(v.levels); i++ {
		lvl := v.levels[i]
		if lvl.XUnits < minX {
			minX = lvl.XUnits
		}
		if lvl.YUnits < minY {
			minY = lvl.YUnits
		}
		if lvl.XUnits+lvl.WidthUnits > maxX {
			maxX = lvl.XUnits + lvl.WidthUnits
		}
		if lvl.YUnits+lvl.HeightUnits > maxY {
			maxY = lvl.YUnits + lvl.HeightUnits
		}
	}
	worldW := float64(maxX-minX) + 8
	worldH := float64(maxY-minY) + 8
	if worldW < 1 {
		worldW = 1
	}
	if worldH < 1 {
		worldH = 1
	}

	targetScaleX := float64(canvasW) / worldW
	targetScaleY := float64(canvasH) / worldH
	targetScale := math.Min(targetScaleX, targetScaleY)
	base := v.unitPixels
	if base <= 0 {
		base = 16
	}
	v.zoom = targetScale / base
	if v.zoom < 0.25 {
		v.zoom = 0.25
	}
	if v.zoom > 3.0 {
		v.zoom = 3.0
	}

	s := v.worldScale()
	v.panX = float64(canvasW)/2 - (float64(minX+maxX)/2)*s
	v.panY = float64(canvasH)/2 - (float64(minY+maxY)/2)*s
	v.framed = true
}

func (v *LevelOverviewView) Update(uiHovered bool) {
	if v == nil || !v.active {
		return
	}
	sw, sh := ebiten.Monitor().Size()
	canvasX, _, canvasW, canvasH := v.canvasBounds(sw, sh)

	if v.dirty {
		v.rebuild()
	}
	if !v.framed {
		v.frameToCanvas(canvasW, canvasH)
	}

	mx, my := ebiten.CursorPosition()
	inCanvas := mx >= canvasX && mx < canvasX+canvasW && my >= 0 && my < canvasH
	v.hovered = -1
	if inCanvas {
		v.hovered = v.levelAtScreenPos(mx, my, canvasX)
	}

	if inCanvas && !uiHovered {
		if _, wy := ebiten.Wheel(); wy != 0 {
			oldScale := v.worldScale()
			if wy > 0 {
				v.zoom *= 1.1
			} else {
				v.zoom /= 1.1
			}
			if v.zoom < 0.15 {
				v.zoom = 0.15
			}
			if v.zoom > 5.0 {
				v.zoom = 5.0
			}
			newScale := v.worldScale()
			if oldScale > 0 && newScale != oldScale {
				wx := (float64(mx-canvasX) - v.panX) / oldScale
				wy2 := (float64(my) - v.panY) / oldScale
				v.panX = float64(mx-canvasX) - wx*newScale
				v.panY = float64(my) - wy2*newScale
			}
		}
	}

	if inCanvas && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonMiddle) {
		v.isPanning = true
		v.lastPanX = mx
		v.lastPanY = my
	}
	if v.isPanning && ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		dx := mx - v.lastPanX
		dy := my - v.lastPanY
		v.panX += float64(dx)
		v.panY += float64(dy)
		v.lastPanX = mx
		v.lastPanY = my
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonMiddle) {
		v.isPanning = false
	}

	if inCanvas && !uiHovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		v.pressedNode = v.levelAtScreenPos(mx, my, canvasX)
		v.pressScreenX = mx
		v.pressScreenY = my
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if v.pressedNode >= 0 && inCanvas {
			releasedNode := v.levelAtScreenPos(mx, my, canvasX)
			dx := mx - v.pressScreenX
			dy := my - v.pressScreenY
			dist := math.Sqrt(float64(dx*dx + dy*dy))
			if releasedNode == v.pressedNode && dist <= 6 {
				if v.pressedNode < len(v.levels) && v.onLoadLevel != nil {
					if err := v.onLoadLevel(v.levels[v.pressedNode].Name); err != nil {
						v.lastError = err.Error()
					} else {
						v.active = false
						v.dirty = true
					}
				}
			}
		}
		v.pressedNode = -1
	}
}

func (v *LevelOverviewView) Draw(screen *ebiten.Image) {
	if v == nil || !v.active {
		return
	}
	sw, sh := screen.Size()
	canvasX, _, canvasW, canvasH := v.canvasBounds(sw, sh)
	ebitenutil.DrawRect(screen, float64(canvasX), 0, float64(canvasW), float64(canvasH), color.RGBA{R: 18, G: 22, B: 30, A: 255})

	if v.dirty {
		v.rebuild()
	}
	if !v.framed {
		v.frameToCanvas(canvasW, canvasH)
	}
	if v.lastError != "" {
		ebitenutil.DebugPrintAt(screen, "Level overview error: "+v.lastError, canvasX+12, 36)
	}

	current := ""
	if v.getCurrentLvl != nil {
		current = normalizeLevelFileName(v.getCurrentLvl())
	}

	for i := range v.levels {
		src := v.levels[i]
		sx0, sy0 := v.worldToScreen(canvasX, float64(src.XUnits), float64(src.YUnits))
		sxCenter := sx0 + float64(src.WidthUnits)*v.worldScale()/2
		syCenter := sy0 + float64(src.HeightUnits)*v.worldScale()/2
		for _, tr := range src.Transitions {
			ti, ok := v.levelIndex[tr.To]
			if !ok {
				continue
			}
			target := v.levels[ti]
			tx0, ty0 := v.worldToScreen(canvasX, float64(target.XUnits), float64(target.YUnits))
			txCenter := tx0 + float64(target.WidthUnits)*v.worldScale()/2
			tyCenter := ty0 + float64(target.HeightUnits)*v.worldScale()/2
			lineCol := color.RGBA{R: 100, G: 140, B: 200, A: 220}
			if len(src.Issues) > 0 || len(target.Issues) > 0 {
				lineCol = color.RGBA{R: 220, G: 90, B: 90, A: 220}
			}
			ebitenutil.DrawLine(screen, sxCenter, syCenter, txCenter, tyCenter, lineCol)
		}
	}

	for i := range v.levels {
		lvl := v.levels[i]
		sx, sy := v.worldToScreen(canvasX, float64(lvl.XUnits), float64(lvl.YUnits))
		swPx := float64(lvl.WidthUnits) * v.worldScale()
		shPx := float64(lvl.HeightUnits) * v.worldScale()
		fill := color.RGBA{R: 58, G: 74, B: 94, A: 255}
		border := color.RGBA{R: 170, G: 205, B: 250, A: 255}
		if lvl.Name == current {
			fill = color.RGBA{R: 99, G: 84, B: 45, A: 255}
			border = color.RGBA{R: 255, G: 218, B: 128, A: 255}
		}
		if len(lvl.Issues) > 0 {
			fill = color.RGBA{R: 120, G: 34, B: 34, A: 255}
			border = color.RGBA{R: 255, G: 130, B: 130, A: 255}
		}
		ebitenutil.DrawRect(screen, sx, sy, swPx, shPx, fill)
		ebitenutil.DrawRect(screen, sx, sy, swPx, 2, border)
		ebitenutil.DrawRect(screen, sx, sy+shPx-2, swPx, 2, border)
		ebitenutil.DrawRect(screen, sx, sy, 2, shPx, border)
		ebitenutil.DrawRect(screen, sx+swPx-2, sy, 2, shPx, border)

		label := strings.TrimSuffix(lvl.Name, ".json")
		ebitenutil.DebugPrintAt(screen, label, int(sx)+6, int(sy)+6)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%dx%d tiles", lvl.WidthTiles, lvl.HeightTiles), int(sx)+6, int(sy)+20)
	}

	if v.hovered >= 0 && v.hovered < len(v.levels) {
		lvl := v.levels[v.hovered]
		if len(lvl.Issues) > 0 {
			mx, my := ebiten.CursorPosition()
			lines := append([]string{"Issues:"}, lvl.Issues...)
			maxLen := 0
			for _, line := range lines {
				if len(line) > maxLen {
					maxLen = len(line)
				}
			}
			boxW := float64(maxLen*7 + 14)
			boxH := float64(len(lines)*14 + 10)
			tx := float64(mx + 14)
			ty := float64(my + 14)
			if tx+boxW > float64(sw) {
				tx = float64(sw) - boxW - 8
			}
			if ty+boxH > float64(sh) {
				ty = float64(sh) - boxH - 8
			}
			ebitenutil.DrawRect(screen, tx, ty, boxW, boxH, color.RGBA{R: 20, G: 20, B: 20, A: 235})
			ebitenutil.DrawRect(screen, tx, ty, boxW, 2, color.RGBA{R: 255, G: 120, B: 120, A: 255})
			for i, line := range lines {
				ebitenutil.DebugPrintAt(screen, line, int(tx)+7, int(ty)+6+i*14)
			}
		}
	}

	ebitenutil.DebugPrintAt(screen, "Level map view: Z toggle, wheel zoom, middle-drag pan, click box to load", canvasX+10, 10)
}
