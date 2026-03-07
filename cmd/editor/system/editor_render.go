package editorsystem

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

type EditorRenderSystem struct {
	assetPaths map[string]string
	images     map[string]*ebiten.Image
}

func NewEditorRenderSystem() *EditorRenderSystem {
	return &EditorRenderSystem{
		assetPaths: make(map[string]string),
		images:     make(map[string]*ebiten.Image),
	}
}

func (s *EditorRenderSystem) Draw(w *ecs.World, screen *ebiten.Image) {
	syncScreenMetrics(w, screen)
	_, session, ok := sessionState(w)
	if !ok {
		return
	}
	_, meta, ok := levelMetaState(w)
	if !ok {
		return
	}
	_, camera, ok := cameraState(w)
	if !ok {
		return
	}
	_, stroke, _ := strokeState(w)
	_, pointer, _ := pointerState(w)
	_, catalog, _ := catalogState(w)
	_, prefabCatalog, _ := prefabCatalogState(w)
	_, placement, _ := prefabPlacementState(w)
	_, selection, _ := entitySelectionState(w)
	_, overview, _ := overviewState(w)

	s.refreshCatalog(catalog)

	screen.Fill(color.RGBA{R: 19, G: 20, B: 26, A: 255})
	if session.OverviewOpen {
		s.drawOverview(screen, camera, overview, session)
		s.drawCanvasOutline(screen, camera)
		s.drawFooter(screen, session, camera)
		return
	}
	// Draw tiles and entities interleaved per layer so tiles above entities
	// can be rendered on top of them (editor should mimic runtime ordering).
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if !layerVisible(layer) {
			continue
		}
		// draw this layer's tiles
		s.drawTilesForLayer(w, screen, meta, camera, entity)
		// draw entities that live on this layer
		s.drawEntitiesOnLayer(screen, camera, prefabCatalog, selection, w, entity)
	}
	if session.PhysicsHighlight {
		s.drawPhysicsHighlight(w, screen, meta, camera)
	}
	s.drawAreaOverlays(screen, camera, selection, w)
	s.drawGrid(screen, meta, camera)
	if pointer != nil && pointer.HasCell && placement != nil && placement.SelectedPath == "" && !session.TransitionMode && !session.GateMode {
		s.drawToolCursorPreview(w, screen, camera, session, meta, pointer, prefabCatalog)
	}
	s.drawPreview(w, screen, camera, meta, stroke, prefabCatalog)
	s.drawCurrentLayerEntityOutlines(screen, camera, prefabCatalog, selection, session, w)
	s.drawCanvasOutline(screen, camera)
	s.drawFooter(screen, session, camera)
	if pointer != nil && pointer.HasCell {
		s.drawHoveredCell(screen, camera, pointer.CellX, pointer.CellY, color.RGBA{R: 255, G: 255, B: 255, A: 140})
	}
}

func (s *EditorRenderSystem) refreshCatalog(catalog *editorcomponent.TilesetCatalog) {
	if catalog == nil {
		return
	}
	for _, asset := range catalog.Assets {
		s.assetPaths[asset.Name] = asset.DiskPath
		s.assetPaths[asset.Relative] = asset.DiskPath
	}
}

func (s *EditorRenderSystem) drawFooter(screen *ebiten.Image, session *editorcomponent.EditorSession, camera *editorcomponent.CanvasCamera) {
	statusY := int(camera.ScreenH) - 22
	if session.Status != "" {
		ebitenutil.DebugPrintAt(screen, session.Status, 16, statusY)
	}
	controls := "Ctrl+B/E/F/L/K tool  Ctrl+Z undo  Ctrl+S save  Q/E layer  N/H/Y/T layer ops  Z overview  Del/Esc clear  F12 quit"
	ebitenutil.DebugPrintAt(screen, controls, int(camera.ScreenW)-len(controls)*7-16, statusY)
}

func (s *EditorRenderSystem) drawToolCursorPreview(w *ecs.World, screen *ebiten.Image, camera *editorcomponent.CanvasCamera, session *editorcomponent.EditorSession, meta *editorcomponent.LevelMeta, pointer *editorcomponent.PointerState, catalog *editorcomponent.PrefabCatalog) {
	if session == nil || pointer == nil || !pointer.HasCell {
		return
	}
	switch session.ActiveTool {
	case editorcomponent.ToolBrush, editorcomponent.ToolFill, editorcomponent.ToolLine:
		s.drawTileCursorPreview(screen, camera, pointer.CellX, pointer.CellY, session.SelectedTile)
	case editorcomponent.ToolSpike:
		rotation := spikeRotationForCell(w, meta, pointer.CellX, pointer.CellY)
		s.drawSpikePreview(screen, camera, pointer.CellX, pointer.CellY, rotation, prefabInfoByPath(catalog, "spike.yaml", "spike"))
	}
}

func (s *EditorRenderSystem) drawAreaOverlays(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, selection *editorcomponent.EntitySelectionState, w *ecs.World) {
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	for index, item := range entities.Items {
		if !entityVisibleOnLayer(w, item) {
			continue
		}
		var fill color.RGBA
		switch {
		case isTransitionEntity(item):
			fill = color.RGBA{R: 80, G: 180, B: 255, A: 36}
		case isGateEntity(item):
			fill = color.RGBA{R: 255, G: 110, B: 110, A: 42}
		default:
			continue
		}
		left, top, width, height := entityRect(item)
		vector.DrawFilledRect(screen, float32(camera.CanvasX+(left-camera.X)*camera.Zoom), float32(camera.CanvasY+(top-camera.Y)*camera.Zoom), float32(width*camera.Zoom), float32(height*camera.Zoom), fill, false)
		if selection != nil && selection.SelectedIndex == index {
			ebitenutil.DebugPrintAt(screen, item.Type, int(camera.CanvasX+(left-camera.X)*camera.Zoom)+4, int(camera.CanvasY+(top-camera.Y)*camera.Zoom)+4)
		}
	}
}

func (s *EditorRenderSystem) drawOverview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, state *editorcomponent.OverviewState, session *editorcomponent.EditorSession) {
	if camera == nil || state == nil {
		return
	}
	vector.DrawFilledRect(screen, float32(camera.CanvasX), float32(camera.CanvasY), float32(camera.CanvasW), float32(camera.CanvasH), color.RGBA{R: 18, G: 20, B: 28, A: 255}, false)
	for _, edge := range state.Edges {
		from := findOverviewNode(state, edge.From)
		to := findOverviewNode(state, edge.To)
		if from == nil || to == nil {
			continue
		}
		clr := color.RGBA{R: 110, G: 140, B: 180, A: 160}
		if edge.Warning {
			clr = color.RGBA{R: 255, G: 170, B: 90, A: 210}
		}
		x1 := camera.CanvasX + ((from.X+from.W/2)-state.PanX)*state.Zoom
		y1 := camera.CanvasY + ((from.Y+from.H/2)-state.PanY)*state.Zoom
		x2 := camera.CanvasX + ((to.X+to.W/2)-state.PanX)*state.Zoom
		y2 := camera.CanvasY + ((to.Y+to.H/2)-state.PanY)*state.Zoom
		vector.StrokeLine(screen, float32(x1), float32(y1), float32(x2), float32(y2), 2, clr, false)
	}
	for _, node := range state.Nodes {
		sx := camera.CanvasX + (node.X-state.PanX)*state.Zoom
		sy := camera.CanvasY + (node.Y-state.PanY)*state.Zoom
		sw := node.W * state.Zoom
		sh := node.H * state.Zoom
		fill := color.RGBA{R: 54, G: 62, B: 84, A: 230}
		outline := color.RGBA{R: 124, G: 141, B: 180, A: 255}
		if len(node.Diagnostics) > 0 {
			fill = color.RGBA{R: 92, G: 54, B: 50, A: 235}
			outline = color.RGBA{R: 255, G: 160, B: 105, A: 255}
		}
		if state.HoveredLevel == node.Level {
			outline = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		vector.DrawFilledRect(screen, float32(sx), float32(sy), float32(sw), float32(sh), fill, false)
		vector.StrokeRect(screen, float32(sx), float32(sy), float32(sw), float32(sh), 2, outline, false)
		ebitenutil.DebugPrintAt(screen, node.DisplayName, int(sx)+10, int(sy)+10)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%s", node.Level), int(sx)+10, int(sy)+30)
		if len(node.Diagnostics) > 0 {
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("issues: %d", len(node.Diagnostics)), int(sx)+10, int(sy)+50)
		}
	}
	if hovered := findOverviewNode(state, state.HoveredLevel); hovered != nil && len(hovered.Diagnostics) > 0 {
		tooltip := strings.Join(hovered.Diagnostics, "\n")
		ebitenutil.DebugPrintAt(screen, tooltip, int(camera.CanvasX)+12, int(camera.CanvasY)+12)
	}
	if session != nil {
		ebitenutil.DebugPrintAt(screen, "Overview: wheel zoom, middle pan, drag boxes, click to load", int(camera.CanvasX)+12, int(camera.CanvasY+camera.CanvasH)-20)
	}
}

func (s *EditorRenderSystem) drawTiles(w *ecs.World, screen *ebiten.Image, meta *editorcomponent.LevelMeta, camera *editorcomponent.CanvasCamera) {
	startX := maxInt(0, int(math.Floor(camera.X/TileSize)))
	startY := maxInt(0, int(math.Floor(camera.Y/TileSize)))
	endX := minInt(meta.Width, int(math.Ceil((camera.X+camera.CanvasW/camera.Zoom)/TileSize))+1)
	endY := minInt(meta.Height, int(math.Ceil((camera.Y+camera.CanvasH/camera.Zoom)/TileSize))+1)

	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if !layerVisible(layer) {
			continue
		}
		for y := startY; y < endY; y++ {
			for x := startX; x < endX; x++ {
				index := cellIndex(meta, x, y)
				usage := layer.TilesetUsage[index]
				if usage == nil || usage.Path == "" {
					continue
				}
				img := s.imageFor(usage.Path)
				if img == nil {
					s.drawFallbackTile(screen, camera, x, y, layer.Tiles[index])
					continue
				}
				tileW := usage.TileW
				tileH := usage.TileH
				if tileW <= 0 {
					tileW = TileSize
				}
				if tileH <= 0 {
					tileH = TileSize
				}
				columns := maxInt(1, img.Bounds().Dx()/tileW)
				srcX := (usage.Index % columns) * tileW
				srcY := (usage.Index / columns) * tileH
				if srcX+tileW > img.Bounds().Dx() || srcY+tileH > img.Bounds().Dy() {
					continue
				}
				sub := img.SubImage(image.Rect(srcX, srcY, srcX+tileW, srcY+tileH)).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(camera.Zoom*float64(TileSize)/float64(tileW), camera.Zoom*float64(TileSize)/float64(tileH))
				op.GeoM.Translate(camera.CanvasX+(float64(x*TileSize)-camera.X)*camera.Zoom, camera.CanvasY+(float64(y*TileSize)-camera.Y)*camera.Zoom)
				screen.DrawImage(sub, op)
			}
		}
	}
}

func (s *EditorRenderSystem) drawTilesForLayer(w *ecs.World, screen *ebiten.Image, meta *editorcomponent.LevelMeta, camera *editorcomponent.CanvasCamera, layerEntity ecs.Entity) {
	if meta == nil || camera == nil {
		return
	}
	startX := maxInt(0, int(math.Floor(camera.X/TileSize)))
	startY := maxInt(0, int(math.Floor(camera.Y/TileSize)))
	endX := minInt(meta.Width, int(math.Ceil((camera.X+camera.CanvasW/camera.Zoom)/TileSize))+1)
	endY := minInt(meta.Height, int(math.Ceil((camera.Y+camera.CanvasH/camera.Zoom)/TileSize))+1)

	layer, _ := ecs.Get(w, layerEntity, editorcomponent.LayerDataComponent.Kind())
	if layer == nil || !layerVisible(layer) {
		return
	}

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			index := cellIndex(meta, x, y)
			usage := layer.TilesetUsage[index]
			if usage == nil || usage.Path == "" {
				continue
			}
			img := s.imageFor(usage.Path)
			if img == nil {
				s.drawFallbackTile(screen, camera, x, y, layer.Tiles[index])
				continue
			}
			tileW := usage.TileW
			tileH := usage.TileH
			if tileW <= 0 {
				tileW = TileSize
			}
			if tileH <= 0 {
				tileH = TileSize
			}
			columns := maxInt(1, img.Bounds().Dx()/tileW)
			srcX := (usage.Index % columns) * tileW
			srcY := (usage.Index / columns) * tileH
			if srcX+tileW > img.Bounds().Dx() || srcY+tileH > img.Bounds().Dy() {
				continue
			}
			sub := img.SubImage(image.Rect(srcX, srcY, srcX+tileW, srcY+tileH)).(*ebiten.Image)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(camera.Zoom*float64(TileSize)/float64(tileW), camera.Zoom*float64(TileSize)/float64(tileH))
			op.GeoM.Translate(camera.CanvasX+(float64(x*TileSize)-camera.X)*camera.Zoom, camera.CanvasY+(float64(y*TileSize)-camera.Y)*camera.Zoom)
			screen.DrawImage(sub, op)
		}
	}
}

func (s *EditorRenderSystem) drawEntitiesOnLayer(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, catalog *editorcomponent.PrefabCatalog, selection *editorcomponent.EntitySelectionState, w *ecs.World, layerEntity ecs.Entity) {
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	thisIdx := layerEntityIndex(w, layerEntity)
	if thisIdx < 0 {
		return
	}
	indices := make([]int, 0, len(entities.Items))
	for index := range entities.Items {
		item := entities.Items[index]
		if normalizedEntityLayerIndex(item) != thisIdx || !entityVisibleOnLayer(w, item) {
			continue
		}
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		return compareEditorEntities(catalog, entities.Items, indices[i], indices[j]) < 0
	})

	previewItem, previewPrefab, previewVisible := selectedPrefabPreview(w)
	if previewVisible && normalizedEntityLayerIndex(previewItem) != thisIdx {
		previewVisible = false
	}
	previewOrder := 0
	if previewVisible {
		previewOrder = editorEntityRenderOrder(catalog, previewItem)
	}
	previewDrawn := false
	for _, index := range indices {
		item := entities.Items[index]
		if previewVisible && !previewDrawn && editorEntityRenderOrder(catalog, item) > previewOrder {
			s.drawPrefabPreview(screen, camera, previewItem, previewPrefab)
			previewDrawn = true
		}
		prefab := prefabInfoForEntity(catalog, item)
		s.drawEntity(screen, camera, item, prefab)
	}
	if previewVisible && !previewDrawn {
		s.drawPrefabPreview(screen, camera, previewItem, previewPrefab)
	}
}

func (s *EditorRenderSystem) drawFallbackTile(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY, value int) {
	if value == 0 {
		return
	}
	shade := uint8(80 + (value*13)%120)
	x := camera.CanvasX + (float64(cellX*TileSize)-camera.X)*camera.Zoom
	y := camera.CanvasY + (float64(cellY*TileSize)-camera.Y)*camera.Zoom
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), color.RGBA{R: shade, G: 120, B: 120, A: 180}, false)
}

func (s *EditorRenderSystem) drawCurrentLayerEntityOutlines(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, catalog *editorcomponent.PrefabCatalog, selection *editorcomponent.EntitySelectionState, session *editorcomponent.EditorSession, w *ecs.World) {
	if session == nil {
		return
	}
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	for _, index := range currentLayerOutlineIndices(w, session, entities.Items) {
		item := entities.Items[index]
		s.drawEntityOutline(screen, camera, item, prefabInfoForEntity(catalog, item), color.RGBA{R: 120, G: 210, B: 255, A: 120})
	}
	if selection != nil {
		if selection.HoveredIndex >= 0 && selection.HoveredIndex < len(entities.Items) && entitySelectableOnCurrentLayer(w, session, entities.Items[selection.HoveredIndex]) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.HoveredIndex], prefabInfoForEntity(catalog, entities.Items[selection.HoveredIndex]), color.RGBA{R: 255, G: 255, B: 255, A: 160})
		}
		if selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) && entitySelectableOnCurrentLayer(w, session, entities.Items[selection.SelectedIndex]) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.SelectedIndex], prefabInfoForEntity(catalog, entities.Items[selection.SelectedIndex]), color.RGBA{R: 255, G: 215, B: 0, A: 220})
		}
	}
}

func currentLayerOutlineIndices(w *ecs.World, session *editorcomponent.EditorSession, items []levels.Entity) []int {
	indices := make([]int, 0, len(items))
	for index := range items {
		if !entitySelectableOnCurrentLayer(w, session, items[index]) {
			continue
		}
		indices = append(indices, index)
	}
	return indices
}

func (s *EditorRenderSystem) drawGrid(screen *ebiten.Image, meta *editorcomponent.LevelMeta, camera *editorcomponent.CanvasCamera) {
	startX := maxInt(0, int(math.Floor(camera.X/TileSize)))
	startY := maxInt(0, int(math.Floor(camera.Y/TileSize)))
	endX := minInt(meta.Width, int(math.Ceil((camera.X+camera.CanvasW/camera.Zoom)/TileSize))+1)
	endY := minInt(meta.Height, int(math.Ceil((camera.Y+camera.CanvasH/camera.Zoom)/TileSize))+1)
	gridColor := color.RGBA{R: 90, G: 95, B: 112, A: 110}
	for x := startX; x <= endX; x++ {
		sx := camera.CanvasX + (float64(x*TileSize)-camera.X)*camera.Zoom
		vector.StrokeLine(screen, float32(sx), float32(camera.CanvasY), float32(sx), float32(camera.CanvasY+camera.CanvasH), 1, gridColor, false)
	}
	for y := startY; y <= endY; y++ {
		sy := camera.CanvasY + (float64(y*TileSize)-camera.Y)*camera.Zoom
		vector.StrokeLine(screen, float32(camera.CanvasX), float32(sy), float32(camera.CanvasX+camera.CanvasW), float32(sy), 1, gridColor, false)
	}
}

func (s *EditorRenderSystem) drawPhysicsHighlight(w *ecs.World, screen *ebiten.Image, meta *editorcomponent.LevelMeta, camera *editorcomponent.CanvasCamera) {
	if meta == nil || camera == nil {
		return
	}
	overlay := color.RGBA{R: 255, G: 96, B: 96, A: 72}
	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if !layerVisible(layer) || !layer.Physics {
			continue
		}
		for index, value := range layer.Tiles {
			if value == 0 {
				continue
			}
			x := index % meta.Width
			y := index / meta.Width
			dx := camera.CanvasX + (float64(x*TileSize)-camera.X)*camera.Zoom
			dy := camera.CanvasY + (float64(y*TileSize)-camera.Y)*camera.Zoom
			vector.DrawFilledRect(screen, float32(dx), float32(dy), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), overlay, false)
		}
	}
}

func (s *EditorRenderSystem) drawEntities(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, catalog *editorcomponent.PrefabCatalog, selection *editorcomponent.EntitySelectionState, w *ecs.World) {
	_, entities, ok := entitiesState(w)
	if !ok || entities == nil {
		return
	}
	indices := make([]int, 0, len(entities.Items))
	for index := range entities.Items {
		indices = append(indices, index)
	}
	sort.SliceStable(indices, func(i, j int) bool {
		return compareEditorEntities(catalog, entities.Items, indices[i], indices[j]) < 0
	})
	for _, index := range indices {
		item := entities.Items[index]
		if !entityVisibleOnLayer(w, item) {
			continue
		}
		prefab := prefabInfoForEntity(catalog, item)
		s.drawEntity(screen, camera, item, prefab)
	}
	if selection != nil {
		if selection.HoveredIndex >= 0 && selection.HoveredIndex < len(entities.Items) && entityVisibleOnLayer(w, entities.Items[selection.HoveredIndex]) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.HoveredIndex], prefabInfoForEntity(catalog, entities.Items[selection.HoveredIndex]), color.RGBA{R: 255, G: 255, B: 255, A: 110})
		}
		if selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) && entityVisibleOnLayer(w, entities.Items[selection.SelectedIndex]) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.SelectedIndex], prefabInfoForEntity(catalog, entities.Items[selection.SelectedIndex]), color.RGBA{R: 255, G: 215, B: 0, A: 220})
		}
	}
}

func (s *EditorRenderSystem) drawEntity(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, item levels.Entity, prefab *editorio.PrefabInfo) {
	prefab = resolvedPrefabInfoForItem(item, prefab)
	if prefab != nil && prefab.Preview.ImagePath != "" {
		img := s.imageFor(prefab.Preview.ImagePath)
		if img != nil {
			frame := image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy())
			if prefab.Preview.FrameW > 0 && prefab.Preview.FrameH > 0 {
				frame = image.Rect(prefab.Preview.FrameX, prefab.Preview.FrameY, prefab.Preview.FrameX+prefab.Preview.FrameW, prefab.Preview.FrameY+prefab.Preview.FrameH)
			}
			if frame.Max.X <= img.Bounds().Dx() && frame.Max.Y <= img.Bounds().Dy() {
				sub := img.SubImage(frame).(*ebiten.Image)
				frameW := float64(frame.Dx())
				frameH := float64(frame.Dy())
				originX, originY := prefabPreviewOrigin(prefab, frameW, frameH)
				anchorX, anchorY := entityAnchorPosition(item, originX, originY)
				scaleX, scaleY := entityPreviewScale(item, prefab)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-originX, -originY)
				op.GeoM.Scale(scaleX, scaleY)
				if rotation := entityRotation(item); rotation != 0 {
					op.GeoM.Rotate(rotation)
				}
				if prefab.Preview.HasTint {
					op.ColorScale.Scale(float32(prefab.Preview.TintR), float32(prefab.Preview.TintG), float32(prefab.Preview.TintB), float32(prefab.Preview.TintA))
				}
				op.GeoM.Scale(camera.Zoom, camera.Zoom)
				op.GeoM.Translate(camera.CanvasX+(anchorX-camera.X)*camera.Zoom, camera.CanvasY+(anchorY-camera.Y)*camera.Zoom)
				screen.DrawImage(sub, op)
				return
			}
		}
	}
	left, top, width, height := entityBounds(item, prefab)
	vector.DrawFilledRect(screen, float32(camera.CanvasX+(left-camera.X)*camera.Zoom), float32(camera.CanvasY+(top-camera.Y)*camera.Zoom), float32(width*camera.Zoom), float32(height*camera.Zoom), fallbackEntityColor(item.Type), false)
}

func (s *EditorRenderSystem) drawEntityOutline(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, item levels.Entity, prefab *editorio.PrefabInfo, clr color.RGBA) {
	left, top, width, height := entityBounds(item, prefab)
	vector.StrokeRect(screen, float32(camera.CanvasX+(left-camera.X)*camera.Zoom), float32(camera.CanvasY+(top-camera.Y)*camera.Zoom), float32(width*camera.Zoom), float32(height*camera.Zoom), 2, clr, false)
}

func (s *EditorRenderSystem) drawPrefabPreview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, item levels.Entity, prefab *editorio.PrefabInfo) {
	if prefab == nil {
		vector.StrokeRect(screen, float32(camera.CanvasX+(float64(item.X)-camera.X)*camera.Zoom), float32(camera.CanvasY+(float64(item.Y)-camera.Y)*camera.Zoom), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), 2, color.RGBA{R: 80, G: 200, B: 255, A: 180}, false)
		return
	}
	s.drawEntity(screen, camera, item, prefab)
	s.drawEntityOutline(screen, camera, item, prefab, color.RGBA{R: 80, G: 200, B: 255, A: 210})
}

func (s *EditorRenderSystem) drawPreview(w *ecs.World, screen *ebiten.Image, camera *editorcomponent.CanvasCamera, meta *editorcomponent.LevelMeta, stroke *editorcomponent.ToolStroke, catalog *editorcomponent.PrefabCatalog) {
	if stroke == nil {
		return
	}
	if stroke.Tool == editorcomponent.ToolSpike {
		prefab := prefabInfoByPath(catalog, "spike.yaml", "spike")
		for _, cell := range stroke.Preview {
			rotation := spikeRotationForCell(w, meta, cell.X, cell.Y)
			s.drawSpikePreview(screen, camera, cell.X, cell.Y, rotation, prefab)
		}
		return
	}
	previewColor := color.RGBA{R: 255, G: 215, B: 0, A: 150}
	if stroke.Tool == editorcomponent.ToolErase {
		previewColor = color.RGBA{R: 255, G: 80, B: 80, A: 150}
	}
	for _, cell := range stroke.Preview {
		s.drawHoveredCell(screen, camera, cell.X, cell.Y, previewColor)
	}
}

func (s *EditorRenderSystem) drawTileCursorPreview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY int, selection model.TileSelection) {
	selection = selection.Normalize()
	if selection.Path == "" {
		return
	}
	img := s.imageFor(selection.Path)
	if img == nil {
		x := camera.CanvasX + (float64(cellX*TileSize)-camera.X)*camera.Zoom
		y := camera.CanvasY + (float64(cellY*TileSize)-camera.Y)*camera.Zoom
		vector.DrawFilledRect(screen, float32(x), float32(y), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), color.RGBA{R: 80, G: 200, B: 255, A: 72}, false)
		return
	}
	tileW := selection.TileW
	tileH := selection.TileH
	if tileW <= 0 {
		tileW = TileSize
	}
	if tileH <= 0 {
		tileH = TileSize
	}
	columns := maxInt(1, img.Bounds().Dx()/tileW)
	srcX := (selection.Index % columns) * tileW
	srcY := (selection.Index / columns) * tileH
	if srcX+tileW > img.Bounds().Dx() || srcY+tileH > img.Bounds().Dy() {
		return
	}
	sub := img.SubImage(image.Rect(srcX, srcY, srcX+tileW, srcY+tileH)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.ColorScale.Scale(1, 1, 1, 0.62)
	op.GeoM.Scale(camera.Zoom*float64(TileSize)/float64(tileW), camera.Zoom*float64(TileSize)/float64(tileH))
	op.GeoM.Translate(camera.CanvasX+(float64(cellX*TileSize)-camera.X)*camera.Zoom, camera.CanvasY+(float64(cellY*TileSize)-camera.Y)*camera.Zoom)
	screen.DrawImage(sub, op)
}

func (s *EditorRenderSystem) drawSpikePreview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY int, rotation float64, prefab *editorio.PrefabInfo) {
	item := levels.Entity{
		Type: "spike",
		X:    cellX * TileSize,
		Y:    cellY * TileSize,
		Props: map[string]interface{}{
			"prefab":   "spike.yaml",
			"rotation": rotation,
		},
	}
	s.drawEntity(screen, camera, item, prefab)
	s.drawEntityOutline(screen, camera, item, prefab, color.RGBA{R: 255, G: 120, B: 80, A: 210})
}

func (s *EditorRenderSystem) drawHoveredCell(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY int, clr color.RGBA) {
	x := camera.CanvasX + (float64(cellX*TileSize)-camera.X)*camera.Zoom
	y := camera.CanvasY + (float64(cellY*TileSize)-camera.Y)*camera.Zoom
	vector.StrokeRect(screen, float32(x), float32(y), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), 2, clr, false)
}

func (s *EditorRenderSystem) drawCanvasOutline(screen *ebiten.Image, camera *editorcomponent.CanvasCamera) {
	vector.StrokeRect(screen, float32(camera.CanvasX), float32(camera.CanvasY), float32(camera.CanvasW), float32(camera.CanvasH), 1, color.RGBA{R: 175, G: 182, B: 198, A: 255}, false)
}

func (s *EditorRenderSystem) imageFor(name string) *ebiten.Image {
	path := s.assetPaths[name]
	if path == "" {
		path = s.assetPaths[filepath.Base(name)]
	}
	if path == "" {
		return nil
	}
	if image, ok := s.images[path]; ok {
		return image
	}
	image, _, err := ebitenutil.NewImageFromFile(path)
	if err != nil {
		s.images[path] = nil
		return nil
	}
	s.images[path] = image
	return image
}

func syncScreenMetrics(w *ecs.World, screen *ebiten.Image) {
	_, camera, ok := cameraState(w)
	if !ok || camera == nil {
		return
	}
	bounds := screen.Bounds()
	camera.ScreenW = float64(bounds.Dx())
	camera.ScreenH = float64(bounds.Dy())
	layoutPanels(camera)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func entityRotation(item levels.Entity) float64 {
	if transformProps := entityComponentOverrideValues(item.Props, "transform"); transformProps != nil {
		if _, ok := transformProps["rotation"]; ok {
			rotation := toFloat(transformProps["rotation"])
			if rotation == 0 {
				return 0
			}
			return rotation * math.Pi / 180.0
		}
	}
	if item.Props == nil {
		return 0
	}
	rotation := toFloat(item.Props["rotation"])
	if rotation == 0 {
		return 0
	}
	return rotation * math.Pi / 180.0
}

func fallbackEntityColor(entityType string) color.RGBA {
	var hash uint8 = 40
	for _, ch := range entityType {
		hash += uint8(ch)
	}
	return color.RGBA{R: 80 + hash%90, G: 110 + hash%80, B: 140 + hash%70, A: 180}
}

func layerEntityIndex(w *ecs.World, target ecs.Entity) int {
	for i, layer := range layerEntities(w) {
		if layer == target {
			return i
		}
	}
	return -1
}

func selectedPrefabPreview(w *ecs.World) (levels.Entity, *editorio.PrefabInfo, bool) {
	_, session, ok := sessionState(w)
	if !ok || session == nil {
		return levels.Entity{}, nil, false
	}
	_, placement, ok := prefabPlacementState(w)
	if !ok || placement == nil || placement.SelectedPath == "" || placement.SelectedType == "" {
		return levels.Entity{}, nil, false
	}
	_, pointer, ok := pointerState(w)
	if !ok || pointer == nil || !pointer.HasCell {
		return levels.Entity{}, nil, false
	}
	_, catalog, _ := prefabCatalogState(w)
	prefab := prefabInfoByPath(catalog, placement.SelectedPath, placement.SelectedType)
	item := levels.Entity{
		Type: placement.SelectedType,
		X:    pointer.CellX * TileSize,
		Y:    pointer.CellY * TileSize,
		Props: map[string]interface{}{
			"layer":  session.CurrentLayer,
			"prefab": placement.SelectedPath,
		},
	}
	if !entityVisibleOnLayer(w, item) {
		return levels.Entity{}, nil, false
	}
	return item, prefab, true
}
