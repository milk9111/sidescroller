package editorsystem

import (
	"fmt"
	"math"
	"sort"
	"strings"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/ecs"
)

type EditorOverviewSystem struct {
	workspaceRoot string
}

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

func NewEditorOverviewSystem(workspaceRoot string) *EditorOverviewSystem {
	return &EditorOverviewSystem{workspaceRoot: workspaceRoot}
}

func (s *EditorOverviewSystem) Update(w *ecs.World) {
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return
	}
	_, state, ok := overviewState(w)
	if !ok || state == nil {
		return
	}
	if state.Zoom <= 0 {
		state.Zoom = 1
	}
	if state.LoadLevel != "" {
		s.loadLevel(w, session, state)
	}
	if !session.OverviewOpen {
		state.HoveredLevel = ""
		state.DraggingLevel = ""
		state.PressedLevel = ""
		state.DragMoved = false
		return
	}

	if state.NeedsRefresh || len(state.Nodes) == 0 {
		s.rebuildGraph(w, session, state)
	}

	_, input, ok := rawInputState(w)
	if !ok || input == nil {
		return
	}
	_, pointer, ok := pointerState(w)
	if !ok || pointer == nil {
		return
	}
	_, camera, ok := cameraState(w)
	if !ok || camera == nil {
		return
	}
	if !pointer.InCanvas {
		state.HoveredLevel = ""
	}

	if pointer.InCanvas && input.WheelY != 0 {
		worldX, worldY := overviewWorldPosition(camera, state, float64(input.MouseX), float64(input.MouseY))
		state.Zoom = clampFloat(state.Zoom+(input.WheelY*0.125), 0.25, 4.0)
		state.PanX = worldX - (float64(input.MouseX)-camera.CanvasX)/state.Zoom
		state.PanY = worldY - (float64(input.MouseY)-camera.CanvasY)/state.Zoom
	}

	if input.MiddleDown && pointer.InCanvas && !state.PanActive {
		state.PanActive = true
		state.PanMouseX = input.MouseX
		state.PanMouseY = input.MouseY
		state.PanStartX = state.PanX
		state.PanStartY = state.PanY
	}
	if input.MiddleJustReleased || !input.MiddleDown {
		state.PanActive = false
	}
	if state.PanActive && input.MiddleDown && pointer.InCanvas && state.PressedLevel == "" && state.DraggingLevel == "" {
		state.PanX = state.PanStartX - float64(input.MouseX-state.PanMouseX)/state.Zoom
		state.PanY = state.PanStartY - float64(input.MouseY-state.PanMouseY)/state.Zoom
	}

	state.HoveredLevel = ""
	if pointer.InCanvas {
		state.HoveredLevel = s.hitNode(camera, state, float64(input.MouseX), float64(input.MouseY))
	}
	if input.LeftJustPressed && pointer.InCanvas && state.HoveredLevel != "" {
		state.PressedLevel = state.HoveredLevel
		state.DraggingLevel = state.HoveredLevel
		worldX, worldY := overviewWorldPosition(camera, state, float64(input.MouseX), float64(input.MouseY))
		if node := findOverviewNode(state, state.DraggingLevel); node != nil {
			state.DragOffsetX = worldX - node.X
			state.DragOffsetY = worldY - node.Y
		}
		state.DragMoved = false
	}
	if state.DraggingLevel != "" && input.LeftDown && pointer.InCanvas {
		worldX, worldY := overviewWorldPosition(camera, state, float64(input.MouseX), float64(input.MouseY))
		if node := findOverviewNode(state, state.DraggingLevel); node != nil {
			nextX := worldX - state.DragOffsetX
			nextY := worldY - state.DragOffsetY
			if math.Abs(nextX-node.X) > 0.5 || math.Abs(nextY-node.Y) > 0.5 {
				state.DragMoved = true
				state.NeedsPersist = true
			}
			node.X = nextX
			node.Y = nextY
			node.HasManual = true
		}
	}
	if state.DraggingLevel != "" && input.LeftJustReleased {
		if !state.DragMoved && state.PressedLevel != "" {
			state.LoadLevel = state.PressedLevel
		}
		state.DraggingLevel = ""
		state.PressedLevel = ""
		if state.NeedsPersist {
			s.persistLayout(session, state)
		}
	}
}

func (s *EditorOverviewSystem) loadLevel(w *ecs.World, session *editorcomponent.EditorSession, state *editorcomponent.OverviewState) {
	levelName := state.LoadLevel
	state.LoadLevel = ""
	doc, normalized, err := editorio.LoadLevel(s.workspaceRoot, levelName)
	if err != nil {
		session.Status = fmt.Sprintf("Overview load failed: %v", err)
		return
	}
	applyLoadedLevel(w, normalized, doc)
	session.OverviewOpen = false
	session.Status = "Loaded " + normalized
}

func (s *EditorOverviewSystem) rebuildGraph(w *ecs.World, session *editorcomponent.EditorSession, state *editorcomponent.OverviewState) {
	records, err := editorio.ScanLevelsForOverview(s.workspaceRoot)
	if err != nil {
		session.Status = fmt.Sprintf("Overview scan failed: %v", err)
		return
	}
	layout, err := editorio.LoadOverviewLayout(s.workspaceRoot)
	if err != nil {
		session.Status = fmt.Sprintf("Overview layout failed: %v", err)
		return
	}
	nodes := make([]editorcomponent.OverviewNode, 0, len(records))
	nodeIndex := make(map[string]int, len(records))
	for index, record := range records {
		nodeW, nodeH := overviewNodeSize(record.Width, record.Height)
		node := editorcomponent.OverviewNode{Level: record.Name, DisplayName: strings.TrimSuffix(record.Name, ".json"), W: nodeW, H: nodeH}
		if entry, ok := layout[record.Name]; ok {
			node.X = entry.X
			node.Y = entry.Y
			node.HasManual = true
		}
		nodeIndex[record.Name] = index
		nodes = append(nodes, node)
	}
	edges := make([]editorcomponent.OverviewEdge, 0)
	placed := make(map[string]bool)
	for _, record := range records {
		seenDir := make(map[string]string)
		for _, transition := range record.Transitions {
			if transition.ToLevel == "" {
				continue
			}
			edge := editorcomponent.OverviewEdge{From: record.Name, To: transition.ToLevel, Direction: transition.EnterDir}
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
	state.Nodes = nodes
	state.Edges = edges
	state.NeedsRefresh = false
}

func overviewNodeSize(levelWidth, levelHeight int) (float64, float64) {
	width := clampFloat(float64(maxInt(1, levelWidth))*overviewNodeTileScale, overviewNodeMinWidth, overviewNodeMaxWidth)
	height := clampFloat(float64(maxInt(1, levelHeight))*overviewNodeTileScale, overviewNodeMinHeight, overviewNodeMaxHeight)
	return width, height
}

func overviewNodePlacement(current, target editorcomponent.OverviewNode, direction string) (float64, float64) {
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

func (s *EditorOverviewSystem) persistLayout(session *editorcomponent.EditorSession, state *editorcomponent.OverviewState) {
	layout := make(map[string]editorio.OverviewLayoutEntry, len(state.Nodes))
	for _, node := range state.Nodes {
		layout[node.Level] = editorio.OverviewLayoutEntry{X: node.X, Y: node.Y}
	}
	if err := editorio.SaveOverviewLayout(s.workspaceRoot, layout); err != nil {
		session.Status = fmt.Sprintf("Overview layout save failed: %v", err)
		return
	}
	state.NeedsPersist = false
	session.Status = "Saved overview layout"
}

func (s *EditorOverviewSystem) hitNode(camera *editorcomponent.CanvasCamera, state *editorcomponent.OverviewState, mouseX, mouseY float64) string {
	for index := len(state.Nodes) - 1; index >= 0; index-- {
		node := state.Nodes[index]
		sx := camera.CanvasX + (node.X-state.PanX)*state.Zoom
		sy := camera.CanvasY + (node.Y-state.PanY)*state.Zoom
		sw := node.W * state.Zoom
		sh := node.H * state.Zoom
		if mouseX >= sx && mouseX <= sx+sw && mouseY >= sy && mouseY <= sy+sh {
			return node.Level
		}
	}
	return ""
}

func overviewWorldPosition(camera *editorcomponent.CanvasCamera, state *editorcomponent.OverviewState, mouseX, mouseY float64) (float64, float64) {
	return state.PanX + (mouseX-camera.CanvasX)/state.Zoom, state.PanY + (mouseY-camera.CanvasY)/state.Zoom
}

func findOverviewNode(state *editorcomponent.OverviewState, level string) *editorcomponent.OverviewNode {
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

func overviewDirectionOffset(direction string) (float64, float64) {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "left":
		return 1, 0
	case "right":
		return -1, 0
	case "up":
		return 0, 1
	case "down":
		return 0, -1
	default:
		return 1, 0
	}
}

var _ ecs.System = (*EditorOverviewSystem)(nil)
