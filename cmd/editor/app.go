package main

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	editorcomponent "github.com/milk9111/sidescroller/cmd/editor/component"
	editorio "github.com/milk9111/sidescroller/cmd/editor/io"
	"github.com/milk9111/sidescroller/cmd/editor/model"
	editorsystem "github.com/milk9111/sidescroller/cmd/editor/system"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/levels"
)

var ErrQuit = errors.New("quit")

type AppConfig struct {
	WorkspaceRoot string
	AssetDir      string
	PrefabDir     string
	LevelDir      string
	LevelName     string
	SaveTarget    string
	Level         *model.LevelDocument
	Assets        []editorio.AssetInfo
	Prefabs       []editorio.PrefabInfo
	AutotileRemap []int
}

type App struct {
	world     *ecs.World
	scheduler *ecs.Scheduler
	render    *editorsystem.EditorRenderSystem
	ui        *editorsystem.EditorUISystem
}

func NewApp(cfg AppConfig) (*App, error) {
	world := ecs.NewWorld()
	scheduler := ecs.NewScheduler()
	render := editorsystem.NewEditorRenderSystem()
	uiSystem, err := editorsystem.NewEditorUISystem(cfg.Assets, cfg.Prefabs)
	if err != nil {
		return nil, err
	}

	if err := bootstrapWorld(world, cfg); err != nil {
		return nil, err
	}

	scheduler.Add(editorsystem.NewEditorInputSystem())
	scheduler.Add(uiSystem)
	scheduler.Add(editorsystem.NewEditorCommandSystem())
	scheduler.Add(editorsystem.NewEditorUndoSystem())
	scheduler.Add(editorsystem.NewEditorCameraSystem())
	scheduler.Add(editorsystem.NewEditorLayerSystem())
	scheduler.Add(editorsystem.NewEditorAreaSystem(cfg.WorkspaceRoot))
	scheduler.Add(editorsystem.NewEditorOverviewSystem(cfg.WorkspaceRoot, cfg.LevelDir))
	scheduler.Add(editorsystem.NewEditorEntitySystem())
	scheduler.Add(editorsystem.NewEditorPrefabSystem(cfg.WorkspaceRoot, cfg.PrefabDir))
	scheduler.Add(editorsystem.NewEditorToolSystem())
	scheduler.Add(editorsystem.NewEditorAutotileSystem())
	scheduler.Add(editorsystem.NewEditorPersistenceSystem(cfg.WorkspaceRoot, cfg.LevelDir))

	return &App{world: world, scheduler: scheduler, render: render, ui: uiSystem}, nil
}

func (a *App) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return ErrQuit
	}
	a.scheduler.Update(a.world)
	if entity, ok := ecs.First(a.world, editorcomponent.EditorSessionComponent.Kind()); ok {
		session, _ := ecs.Get(a.world, entity, editorcomponent.EditorSessionComponent.Kind())
		if session != nil && session.QuitRequested {
			return ErrQuit
		}
	}
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	a.render.Draw(a.world, screen)
	if a.ui != nil {
		a.ui.Draw(screen)
	}
}

func (a *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func bootstrapWorld(world *ecs.World, cfg AppConfig) error {
	doc := cfg.Level
	if doc == nil {
		doc = model.NewLevelDocument(40, 22)
	}
	assetNames := make([]string, 0, len(cfg.Assets))
	for _, asset := range cfg.Assets {
		assetNames = append(assetNames, asset.Name)
	}
	selection := model.InferSelection(doc, assetNames)

	sessionEntity := ecs.CreateEntity(world)
	_ = ecs.Add(world, sessionEntity, editorcomponent.EditorSessionComponent.Kind(), &editorcomponent.EditorSession{
		ActiveTool:   editorcomponent.ToolBrush,
		CurrentLayer: 0,
		SaveTarget:   cfg.SaveTarget,
		AssetDir:     cfg.AssetDir,
		LoadedLevel:  cfg.LevelName,
		SelectedTile: selection,
		Status:       "Ready",
	})
	_ = ecs.Add(world, sessionEntity, editorcomponent.EditorFocusComponent.Kind(), &editorcomponent.EditorFocus{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.EditorClockComponent.Kind(), &editorcomponent.EditorClock{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.LevelMetaComponent.Kind(), &editorcomponent.LevelMeta{
		Width:       doc.Width,
		Height:      doc.Height,
		LoadedLevel: cfg.LevelName,
	})
	_ = ecs.Add(world, sessionEntity, editorcomponent.LevelEntitiesComponent.Kind(), &editorcomponent.LevelEntities{Items: cloneEntities(doc.Entities)})
	_ = ecs.Add(world, sessionEntity, editorcomponent.TilesetCatalogComponent.Kind(), &editorcomponent.TilesetCatalog{Assets: cfg.Assets})
	_ = ecs.Add(world, sessionEntity, editorcomponent.PrefabCatalogComponent.Kind(), &editorcomponent.PrefabCatalog{Items: append([]editorio.PrefabInfo(nil), cfg.Prefabs...)})
	_ = ecs.Add(world, sessionEntity, editorcomponent.PrefabPlacementComponent.Kind(), &editorcomponent.PrefabPlacementState{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.EntitySelectionComponent.Kind(), &editorcomponent.EntitySelectionState{SelectedIndex: -1, HoveredIndex: -1})
	_ = ecs.Add(world, sessionEntity, editorcomponent.EntityClipboardComponent.Kind(), &editorcomponent.EntityClipboardState{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.RawInputStateComponent.Kind(), &editorcomponent.RawInputState{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.PointerStateComponent.Kind(), &editorcomponent.PointerState{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.CanvasCameraComponent.Kind(), &editorcomponent.CanvasCamera{Zoom: 1})
	_ = ecs.Add(world, sessionEntity, editorcomponent.ToolStrokeComponent.Kind(), &editorcomponent.ToolStroke{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.MoveSelectionComponent.Kind(), &editorcomponent.MoveSelectionState{})
	_ = ecs.Add(world, sessionEntity, editorcomponent.AreaDragStateComponent.Kind(), &editorcomponent.AreaDragState{EntityIndex: -1})
	_ = ecs.Add(world, sessionEntity, editorcomponent.UndoStackComponent.Kind(), &editorcomponent.UndoStack{Max: 100})
	_ = ecs.Add(world, sessionEntity, editorcomponent.EditorActionsComponent.Kind(), &editorcomponent.EditorActions{SelectLayer: -1})
	_ = ecs.Add(world, sessionEntity, editorcomponent.AutotileStateComponent.Kind(), &editorcomponent.AutotileState{
		Remap:       append([]int(nil), cfg.AutotileRemap...),
		DirtyCells:  make(map[int]map[int]struct{}),
		FullRebuild: make(map[int]bool),
	})
	_ = ecs.Add(world, sessionEntity, editorcomponent.OverviewStateComponent.Kind(), &editorcomponent.OverviewState{Zoom: 1, NeedsRefresh: true})

	for index, layer := range doc.Layers {
		entity := ecs.CreateEntity(world)
		_ = ecs.Add(world, entity, editorcomponent.LayerDataComponent.Kind(), &editorcomponent.LayerData{
			Name:         layer.Name,
			Order:        index,
			Physics:      layer.Physics,
			Hidden:       false,
			Tiles:        append([]int(nil), layer.Tiles...),
			TilesetUsage: cloneUsage(layer.TilesetUsage),
		})
	}
	return nil
}

func cloneEntities(items []levels.Entity) []levels.Entity {
	cloned := make([]levels.Entity, 0, len(items))
	for _, item := range items {
		copied := item
		if item.Props != nil {
			copied.Props = make(map[string]interface{}, len(item.Props))
			for key, value := range item.Props {
				copied.Props[key] = value
			}
		}
		if copied.Props == nil {
			copied.Props = make(map[string]interface{})
		}
		if _, ok := copied.Props["layer"]; !ok {
			copied.Props["layer"] = 0
		}
		cloned = append(cloned, copied)
	}
	return cloned
}

func cloneUsage(input []*levels.TileInfo) []*levels.TileInfo {
	output := make([]*levels.TileInfo, len(input))
	for index, item := range input {
		if item == nil {
			continue
		}
		copied := *item
		output[index] = &copied
	}
	return output
}
