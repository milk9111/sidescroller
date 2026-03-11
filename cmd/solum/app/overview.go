package app

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"sort"
	"strings"

	g "github.com/AllenDang/giu"
	coreio "github.com/milk9111/sidescroller/internal/editorcore/io"
)

const (
	overviewNodeTileScale = 6.0
	overviewNodeMinWidth  = 72.0
	overviewNodeMinHeight = 72.0
	overviewNodeMaxWidth  = 360.0
	overviewNodeMaxHeight = 260.0
	overviewNodeGapX      = 64.0
	overviewNodeGapY      = 56.0
	overviewWrapWidth     = 960.0
)

func (s *State) toggleOverview() {
	if s == nil {
		return
	}
	s.Overview.Open = !s.Overview.Open
	s.clearOverviewInteractionState()
	if s.Overview.Open {
		s.Overview.NeedsRefresh = true
		s.Status = "Overview opened"
		return
	}
	s.Status = "Overview closed"
}

func (s *State) refreshOverview() {
	if s == nil {
		return
	}
	s.Overview.NeedsRefresh = true
	if s.Overview.Open {
		s.Status = "Overview refresh queued"
	}
}

func (s *State) UpdateOverview(rect image.Rectangle, input ViewportInput) {
	if s == nil {
		return
	}
	if s.Overview.Zoom <= 0 {
		s.Overview.Zoom = 1
	}
	if !s.Overview.Open {
		s.clearOverviewInteractionState()
		return
	}
	if s.Overview.NeedsRefresh || len(s.Overview.Nodes) == 0 {
		s.rebuildOverviewGraph()
	}
	overview := &s.Overview
	if input.WheelY != 0 && pointInRect(rect, input.MouseX, input.MouseY) {
		worldX, worldY := s.overviewWorldPosition(rect, float64(input.MouseX), float64(input.MouseY))
		overview.Zoom = clampFloat(overview.Zoom+(input.WheelY*0.125), 0.25, 4.0)
		overview.PanX = worldX - (float64(input.MouseX)-float64(rect.Min.X))/overview.Zoom
		overview.PanY = worldY - (float64(input.MouseY)-float64(rect.Min.Y))/overview.Zoom
	}
	if input.MiddleDown && pointInRect(rect, input.MouseX, input.MouseY) && !overview.PanActive {
		overview.PanActive = true
		overview.PanMouseX = input.MouseX
		overview.PanMouseY = input.MouseY
		overview.PanStartX = overview.PanX
		overview.PanStartY = overview.PanY
	}
	if input.MiddleJustReleased || !input.MiddleDown {
		overview.PanActive = false
	}
	if overview.PanActive && input.MiddleDown && pointInRect(rect, input.MouseX, input.MouseY) && overview.PressedLevel == "" && overview.DraggingLevel == "" {
		overview.PanX = overview.PanStartX - float64(input.MouseX-overview.PanMouseX)/overview.Zoom
		overview.PanY = overview.PanStartY - float64(input.MouseY-overview.PanMouseY)/overview.Zoom
	}

	overview.HoveredLevel = ""
	if pointInRect(rect, input.MouseX, input.MouseY) {
		overview.HoveredLevel = s.hitOverviewNode(rect, float64(input.MouseX), float64(input.MouseY))
	}
	if input.LeftJustPressed && pointInRect(rect, input.MouseX, input.MouseY) && overview.HoveredLevel != "" {
		overview.PressedLevel = overview.HoveredLevel
		overview.DraggingLevel = overview.HoveredLevel
		worldX, worldY := s.overviewWorldPosition(rect, float64(input.MouseX), float64(input.MouseY))
		if node := findOverviewNode(overview, overview.DraggingLevel); node != nil {
			overview.DragOffsetX = worldX - node.X
			overview.DragOffsetY = worldY - node.Y
		}
		overview.DragMoved = false
	}
	if overview.DraggingLevel != "" && input.LeftDown && pointInRect(rect, input.MouseX, input.MouseY) {
		worldX, worldY := s.overviewWorldPosition(rect, float64(input.MouseX), float64(input.MouseY))
		if node := findOverviewNode(overview, overview.DraggingLevel); node != nil {
			nextX := worldX - overview.DragOffsetX
			nextY := worldY - overview.DragOffsetY
			if math.Abs(nextX-node.X) > 0.5 || math.Abs(nextY-node.Y) > 0.5 {
				overview.DragMoved = true
				overview.NeedsPersist = true
			}
			node.X = nextX
			node.Y = nextY
			node.HasManual = true
		}
	}
	if overview.DraggingLevel != "" && input.LeftJustReleased {
		pressed := overview.PressedLevel
		dragMoved := overview.DragMoved
		persist := overview.NeedsPersist
		s.clearOverviewInteractionState()
		if persist {
			s.persistOverviewLayout()
		}
		if !dragMoved && strings.TrimSpace(pressed) != "" {
			s.loadLevel(pressed)
			if !strings.HasPrefix(s.Status, "Load failed:") {
				s.Overview.Open = false
			}
		}
	}
}

func (s *State) rebuildOverviewGraph() {
	records, err := coreio.ScanLevelsForOverview(s.WorkspaceRoot)
	if err != nil {
		s.Status = fmt.Sprintf("Overview scan failed: %v", err)
		return
	}
	layout, err := coreio.LoadOverviewLayout(s.WorkspaceRoot)
	if err != nil {
		s.Status = fmt.Sprintf("Overview layout failed: %v", err)
		return
	}
	nodes := make([]OverviewNode, 0, len(records))
	nodeIndex := make(map[string]int, len(records))
	for index, record := range records {
		nodeW, nodeH := overviewNodeSize(record.Width, record.Height)
		node := OverviewNode{Level: record.Name, DisplayName: strings.TrimSuffix(record.Name, ".json"), W: nodeW, H: nodeH}
		if entry, ok := layout[record.Name]; ok {
			node.X = entry.X
			node.Y = entry.Y
			node.HasManual = true
		}
		nodeIndex[record.Name] = index
		nodes = append(nodes, node)
	}
	edges := make([]OverviewEdge, 0)
	placed := make(map[string]bool)
	for _, record := range records {
		seenDir := make(map[string]string)
		for _, transition := range record.Transitions {
			if transition.ToLevel == "" {
				continue
			}
			edge := OverviewEdge{From: record.Name, To: transition.ToLevel, Direction: transition.EnterDir}
			if _, ok := nodeIndex[transition.ToLevel]; !ok {
				edge.Warning = true
				nodes[nodeIndex[record.Name]].Diagnostics = append(nodes[nodeIndex[record.Name]].Diagnostics, fmt.Sprintf("Missing target %s", transition.ToLevel))
			} else if previous, exists := seenDir[transition.EnterDir]; exists && previous != transition.ToLevel {
				edge.Warning = true
				nodes[nodeIndex[record.Name]].Diagnostics = append(nodes[nodeIndex[record.Name]].Diagnostics, fmt.Sprintf("Multiple %s exits (%s, %s)", transition.EnterDir, previous, transition.ToLevel))
			} else {
				seenDir[transition.EnterDir] = transition.ToLevel
			}
			edges = append(edges, edge)
		}
	}
	incoming := make(map[string]map[string][]string)
	for _, edge := range edges {
		if _, ok := nodeIndex[edge.To]; !ok {
			continue
		}
		if incoming[edge.To] == nil {
			incoming[edge.To] = make(map[string][]string)
		}
		incoming[edge.To][edge.Direction] = append(incoming[edge.To][edge.Direction], edge.From)
	}
	for levelName, byDir := range incoming {
		for dir, sources := range byDir {
			if len(sources) <= 1 {
				continue
			}
			sort.Strings(sources)
			nodes[nodeIndex[levelName]].Diagnostics = append(nodes[nodeIndex[levelName]].Diagnostics, fmt.Sprintf("Conflicting %s entries from %s", dir, strings.Join(sources, ", ")))
		}
	}
	if len(nodes) > 0 {
		queue := []string{nodes[0].Level}
		placed[nodes[0].Level] = true
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			currentNode := nodes[nodeIndex[current]]
			for _, edge := range edges {
				if edge.From != current {
					continue
				}
				index, ok := nodeIndex[edge.To]
				if !ok || nodes[index].HasManual || placed[edge.To] {
					continue
				}
				nodes[index].X, nodes[index].Y = overviewNodePlacement(currentNode, nodes[index], edge.Direction)
				placed[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	fallbackX := 0.0
	fallbackY := 0.0
	rowHeight := 0.0
	for index := range nodes {
		if nodes[index].HasManual || placed[nodes[index].Level] {
			continue
		}
		if fallbackX > 0 && fallbackX+nodes[index].W > overviewWrapWidth {
			fallbackX = 0
			fallbackY += rowHeight + overviewNodeGapY
			rowHeight = 0
		}
		nodes[index].X = fallbackX
		nodes[index].Y = fallbackY
		fallbackX += nodes[index].W + overviewNodeGapX
		if nodes[index].H > rowHeight {
			rowHeight = nodes[index].H
		}
	}
	s.Overview.Nodes = nodes
	s.Overview.Edges = edges
	s.Overview.NeedsRefresh = false
	if s.Overview.Open {
		s.Status = fmt.Sprintf("Overview ready: %d levels", len(nodes))
	}
}

func (s *State) persistOverviewLayout() {
	layout := make(map[string]coreio.OverviewLayoutEntry, len(s.Overview.Nodes))
	for _, node := range s.Overview.Nodes {
		layout[node.Level] = coreio.OverviewLayoutEntry{X: node.X, Y: node.Y}
	}
	if err := coreio.SaveOverviewLayout(s.WorkspaceRoot, layout); err != nil {
		s.Status = fmt.Sprintf("Overview layout save failed: %v", err)
		return
	}
	s.Overview.NeedsPersist = false
	s.Status = "Saved overview layout"
}

func (s *State) drawOverview(canvas *g.Canvas, rect image.Rectangle) {
	overview := &s.Overview
	canvas.AddRectFilled(rect.Min, rect.Max, color.RGBA{R: 18, G: 20, B: 28, A: 255}, 0, 0)
	canvas.AddRect(rect.Min, rect.Max, color.RGBA{R: 90, G: 105, B: 132, A: 255}, 0, 0, 1)
	for _, edge := range overview.Edges {
		from := findOverviewNode(overview, edge.From)
		to := findOverviewNode(overview, edge.To)
		if from == nil || to == nil {
			continue
		}
		clr := color.RGBA{R: 110, G: 140, B: 180, A: 160}
		if edge.Warning {
			clr = color.RGBA{R: 255, G: 170, B: 90, A: 210}
		}
		x1 := int(float64(rect.Min.X) + ((from.X + from.W/2) - overview.PanX) * overview.Zoom)
		y1 := int(float64(rect.Min.Y) + ((from.Y + from.H/2) - overview.PanY) * overview.Zoom)
		x2 := int(float64(rect.Min.X) + ((to.X + to.W/2) - overview.PanX) * overview.Zoom)
		y2 := int(float64(rect.Min.Y) + ((to.Y + to.H/2) - overview.PanY) * overview.Zoom)
		canvas.AddLine(image.Pt(x1, y1), image.Pt(x2, y2), clr, 2)
	}
	for _, node := range overview.Nodes {
		sx := int(float64(rect.Min.X) + (node.X-overview.PanX)*overview.Zoom)
		sy := int(float64(rect.Min.Y) + (node.Y-overview.PanY)*overview.Zoom)
		sw := int(node.W * overview.Zoom)
		sh := int(node.H * overview.Zoom)
		fill := color.RGBA{R: 54, G: 62, B: 84, A: 230}
		outline := color.RGBA{R: 124, G: 141, B: 180, A: 255}
		if strings.EqualFold(node.Level, s.LoadedLevel) {
			fill = color.RGBA{R: 52, G: 84, B: 72, A: 236}
			outline = color.RGBA{R: 128, G: 214, B: 166, A: 255}
		}
		if len(node.Diagnostics) > 0 {
			fill = color.RGBA{R: 92, G: 54, B: 50, A: 235}
			outline = color.RGBA{R: 255, G: 160, B: 105, A: 255}
		}
		if overview.HoveredLevel == node.Level {
			outline = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		canvas.AddRectFilled(image.Pt(sx, sy), image.Pt(sx+sw, sy+sh), fill, 8, 0)
		canvas.AddRect(image.Pt(sx, sy), image.Pt(sx+sw, sy+sh), outline, 8, 0, 2)
		canvas.AddText(image.Pt(sx+10, sy+10), color.RGBA{R: 242, G: 245, B: 248, A: 255}, node.DisplayName)
		canvas.AddText(image.Pt(sx+10, sy+28), color.RGBA{R: 176, G: 184, B: 196, A: 255}, node.Level)
		if len(node.Diagnostics) > 0 {
			canvas.AddText(image.Pt(sx+10, sy+46), color.RGBA{R: 255, G: 195, B: 148, A: 255}, fmt.Sprintf("issues: %d", len(node.Diagnostics)))
		}
	}
	canvas.AddText(rect.Min.Add(image.Pt(12, 12)), color.RGBA{R: 231, G: 236, B: 244, A: 255}, fmt.Sprintf("Overview  Zoom %.2fx  Levels %d  Edges %d", overview.Zoom, len(overview.Nodes), len(overview.Edges)))
	if hovered := findOverviewNode(overview, overview.HoveredLevel); hovered != nil && len(hovered.Diagnostics) > 0 {
		for index, line := range hovered.Diagnostics {
			canvas.AddText(rect.Min.Add(image.Pt(12, 34+(index*18))), color.RGBA{R: 255, G: 200, B: 168, A: 255}, line)
		}
	}
	canvas.AddText(image.Pt(rect.Min.X+12, rect.Max.Y-22), color.RGBA{R: 170, G: 182, B: 198, A: 255}, "Overview: wheel zoom, middle pan, drag nodes, click to load")
}

func (s *State) hitOverviewNode(rect image.Rectangle, mouseX, mouseY float64) string {
	for index := len(s.Overview.Nodes) - 1; index >= 0; index-- {
		node := s.Overview.Nodes[index]
		sx := float64(rect.Min.X) + (node.X-s.Overview.PanX)*s.Overview.Zoom
		sy := float64(rect.Min.Y) + (node.Y-s.Overview.PanY)*s.Overview.Zoom
		sw := node.W * s.Overview.Zoom
		sh := node.H * s.Overview.Zoom
		if mouseX >= sx && mouseX <= sx+sw && mouseY >= sy && mouseY <= sy+sh {
			return node.Level
		}
	}
	return ""
}

func (s *State) overviewWorldPosition(rect image.Rectangle, mouseX, mouseY float64) (float64, float64) {
	return s.Overview.PanX + (mouseX-float64(rect.Min.X))/s.Overview.Zoom, s.Overview.PanY + (mouseY-float64(rect.Min.Y))/s.Overview.Zoom
}

func (s *State) clearOverviewInteractionState() {
	s.Overview.HoveredLevel = ""
	s.Overview.DraggingLevel = ""
	s.Overview.PressedLevel = ""
	s.Overview.DragOffsetX = 0
	s.Overview.DragOffsetY = 0
	s.Overview.DragMoved = false
	s.Overview.PanActive = false
	if !s.Overview.Open {
		s.Overview.LoadLevel = ""
	}
}

func pointInRect(rect image.Rectangle, mouseX, mouseY int) bool {
	return mouseX >= rect.Min.X && mouseX < rect.Max.X && mouseY >= rect.Min.Y && mouseY < rect.Max.Y
}

func overviewNodeSize(levelWidth, levelHeight int) (float64, float64) {
	width := clampFloat(float64(maxInt(1, levelWidth))*overviewNodeTileScale, overviewNodeMinWidth, overviewNodeMaxWidth)
	height := clampFloat(float64(maxInt(1, levelHeight))*overviewNodeTileScale, overviewNodeMinHeight, overviewNodeMaxHeight)
	return width, height
}

func overviewNodePlacement(current, target OverviewNode, direction string) (float64, float64) {
	x := current.X
	y := current.Y
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "left":
		x += current.W + overviewNodeGapX
	case "right":
		x -= target.W + overviewNodeGapX
	case "up":
		y += current.H + overviewNodeGapY
	case "down":
		y -= target.H + overviewNodeGapY
	default:
		x += current.W + overviewNodeGapX
	}
	return x, y
}

func findOverviewNode(state *OverviewState, level string) *OverviewNode {
	if state == nil {
		return nil
	}
	for index := range state.Nodes {
		if state.Nodes[index].Level == level {
			return &state.Nodes[index]
		}
	}
	return nil
}