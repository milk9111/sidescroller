package editorsystem

import (
	"image"
	"image/color"
	_ "image/png"
	"math"
	"path/filepath"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
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

	s.refreshCatalog(catalog)

	screen.Fill(color.RGBA{R: 19, G: 20, B: 26, A: 255})
	s.drawTiles(w, screen, meta, camera)
	if session.PhysicsHighlight {
		s.drawPhysicsHighlight(w, screen, meta, camera)
	}
	s.drawEntities(screen, camera, prefabCatalog, selection, w)
	s.drawGrid(screen, meta, camera)
	if placement != nil && placement.SelectedPath != "" && pointer != nil && pointer.HasCell {
		s.drawPrefabPreview(screen, camera, pointer.CellX, pointer.CellY, prefabInfoByPath(prefabCatalog, placement.SelectedPath, placement.SelectedType))
	}
	s.drawPreview(screen, camera, stroke)
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
	controls := "Ctrl+B/E/F/L/K tool  Ctrl+Z undo  Ctrl+S save  Q/E layer  N/H/Y layer ops  Del/Esc entity  F12 quit"
	ebitenutil.DebugPrintAt(screen, controls, int(camera.ScreenW)-len(controls)*7-16, statusY)
}

func (s *EditorRenderSystem) drawTiles(w *ecs.World, screen *ebiten.Image, meta *editorcomponent.LevelMeta, camera *editorcomponent.CanvasCamera) {
	startX := maxInt(0, int(math.Floor(camera.X/TileSize)))
	startY := maxInt(0, int(math.Floor(camera.Y/TileSize)))
	endX := minInt(meta.Width, int(math.Ceil((camera.X+camera.CanvasW/camera.Zoom)/TileSize))+1)
	endY := minInt(meta.Height, int(math.Ceil((camera.Y+camera.CanvasH/camera.Zoom)/TileSize))+1)

	for _, entity := range layerEntities(w) {
		layer, _ := ecs.Get(w, entity, editorcomponent.LayerDataComponent.Kind())
		if layer == nil {
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

func (s *EditorRenderSystem) drawFallbackTile(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY, value int) {
	if value == 0 {
		return
	}
	shade := uint8(80 + (value*13)%120)
	x := camera.CanvasX + (float64(cellX*TileSize)-camera.X)*camera.Zoom
	y := camera.CanvasY + (float64(cellY*TileSize)-camera.Y)*camera.Zoom
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), color.RGBA{R: shade, G: 120, B: 120, A: 180}, false)
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
		if layer == nil || !layer.Physics {
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
		left := prefabInfoForEntity(catalog, entities.Items[indices[i]])
		right := prefabInfoForEntity(catalog, entities.Items[indices[j]])
		leftLayer := 0
		rightLayer := 0
		if left != nil {
			leftLayer = left.Preview.RenderLayer
		}
		if right != nil {
			rightLayer = right.Preview.RenderLayer
		}
		if leftLayer == rightLayer {
			return indices[i] < indices[j]
		}
		return leftLayer < rightLayer
	})
	for _, index := range indices {
		item := entities.Items[index]
		prefab := prefabInfoForEntity(catalog, item)
		s.drawEntity(screen, camera, item, prefab)
	}
	if selection != nil {
		if selection.HoveredIndex >= 0 && selection.HoveredIndex < len(entities.Items) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.HoveredIndex], prefabInfoForEntity(catalog, entities.Items[selection.HoveredIndex]), color.RGBA{R: 255, G: 255, B: 255, A: 110})
		}
		if selection.SelectedIndex >= 0 && selection.SelectedIndex < len(entities.Items) {
			s.drawEntityOutline(screen, camera, entities.Items[selection.SelectedIndex], prefabInfoForEntity(catalog, entities.Items[selection.SelectedIndex]), color.RGBA{R: 255, G: 215, B: 0, A: 220})
		}
	}
}

func (s *EditorRenderSystem) drawEntity(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, item levels.Entity, prefab *editorio.PrefabInfo) {
	if prefab != nil && prefab.Preview.ImagePath != "" {
		img := s.imageFor(prefab.Preview.ImagePath)
		if img != nil {
			frame := image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy())
			if prefab.Preview.FrameW > 0 && prefab.Preview.FrameH > 0 {
				frame = image.Rect(prefab.Preview.FrameX, prefab.Preview.FrameY, prefab.Preview.FrameX+prefab.Preview.FrameW, prefab.Preview.FrameY+prefab.Preview.FrameH)
			}
			if frame.Max.X <= img.Bounds().Dx() && frame.Max.Y <= img.Bounds().Dy() {
				sub := img.SubImage(frame).(*ebiten.Image)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(-prefab.Preview.OriginX, -prefab.Preview.OriginY)
				if rotation := entityRotation(item); rotation != 0 {
					op.GeoM.Rotate(rotation)
				}
				op.GeoM.Scale(camera.Zoom, camera.Zoom)
				op.GeoM.Translate(camera.CanvasX+(float64(item.X)-camera.X)*camera.Zoom, camera.CanvasY+(float64(item.Y)-camera.Y)*camera.Zoom)
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

func (s *EditorRenderSystem) drawPrefabPreview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, cellX, cellY int, prefab *editorio.PrefabInfo) {
	item := levels.Entity{Type: "preview", X: cellX * TileSize, Y: cellY * TileSize}
	if prefab == nil {
		vector.StrokeRect(screen, float32(camera.CanvasX+(float64(item.X)-camera.X)*camera.Zoom), float32(camera.CanvasY+(float64(item.Y)-camera.Y)*camera.Zoom), float32(camera.Zoom*TileSize), float32(camera.Zoom*TileSize), 2, color.RGBA{R: 80, G: 200, B: 255, A: 180}, false)
		return
	}
	s.drawEntity(screen, camera, item, prefab)
	s.drawEntityOutline(screen, camera, item, prefab, color.RGBA{R: 80, G: 200, B: 255, A: 210})
}

func (s *EditorRenderSystem) drawPreview(screen *ebiten.Image, camera *editorcomponent.CanvasCamera, stroke *editorcomponent.ToolStroke) {
	if stroke == nil {
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
