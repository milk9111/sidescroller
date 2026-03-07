package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/milk9111/sidescroller/levels"
)

const DefaultTileSize = 32

type TileSelection struct {
	Path  string
	Index int
	TileW int
	TileH int
}

type Layer struct {
	Name         string
	Physics      bool
	Tiles        []int
	TilesetUsage []*levels.TileInfo
}

type LevelDocument struct {
	Width    int
	Height   int
	Layers   []Layer
	Entities []levels.Entity
}

type Snapshot struct {
	Level         LevelDocument
	CurrentLayer  int
	SaveTarget    string
	LoadedLevel   string
	SelectedTile  TileSelection
	StatusMessage string
}

func NewLevelDocument(width, height int) *LevelDocument {
	cellCount := width * height
	return &LevelDocument{
		Width:  width,
		Height: height,
		Layers: []Layer{
			{
				Name:         "Background",
				Physics:      false,
				Tiles:        make([]int, cellCount),
				TilesetUsage: make([]*levels.TileInfo, cellCount),
			},
			{
				Name:         "Physics",
				Physics:      true,
				Tiles:        make([]int, cellCount),
				TilesetUsage: make([]*levels.TileInfo, cellCount),
			},
		},
	}
}

func FromRuntimeLevel(level *levels.Level) *LevelDocument {
	if level == nil {
		return NewLevelDocument(40, 22)
	}

	doc := &LevelDocument{
		Width:    level.Width,
		Height:   level.Height,
		Layers:   make([]Layer, 0, len(level.Layers)),
		Entities: cloneEntities(level.Entities),
	}

	for index, tiles := range level.Layers {
		layer := Layer{
			Name:         defaultLayerName(index, len(level.Layers)),
			Tiles:        append([]int(nil), tiles...),
			TilesetUsage: cloneTileUsageSlice(tilesetUsageAt(level.TilesetUsage, index)),
		}
		if index < len(level.LayerMeta) {
			layer.Physics = level.LayerMeta[index].Physics
			if strings.TrimSpace(level.LayerMeta[index].Name) != "" {
				layer.Name = level.LayerMeta[index].Name
			}
		}
		if layer.TilesetUsage == nil {
			layer.TilesetUsage = make([]*levels.TileInfo, len(layer.Tiles))
		}
		doc.Layers = append(doc.Layers, layer)
	}

	if len(doc.Layers) == 0 {
		return NewLevelDocument(level.Width, level.Height)
	}

	return doc
}

func (d *LevelDocument) Clone() LevelDocument {
	if d == nil {
		return *NewLevelDocument(40, 22)
	}

	clone := LevelDocument{
		Width:    d.Width,
		Height:   d.Height,
		Layers:   make([]Layer, 0, len(d.Layers)),
		Entities: cloneEntities(d.Entities),
	}
	for _, layer := range d.Layers {
		clone.Layers = append(clone.Layers, Layer{
			Name:         layer.Name,
			Physics:      layer.Physics,
			Tiles:        append([]int(nil), layer.Tiles...),
			TilesetUsage: cloneTileUsageSlice(layer.TilesetUsage),
		})
	}
	return clone
}

func (d *LevelDocument) ToRuntimeLevel() *levels.Level {
	if d == nil {
		return nil
	}

	level := &levels.Level{
		Width:        d.Width,
		Height:       d.Height,
		Layers:       make([][]int, 0, len(d.Layers)),
		TilesetUsage: make([][]*levels.TileInfo, 0, len(d.Layers)),
		LayerMeta:    make([]levels.LayerMeta, 0, len(d.Layers)),
		Entities:     cloneEntities(d.Entities),
	}

	for _, layer := range d.Layers {
		level.Layers = append(level.Layers, append([]int(nil), layer.Tiles...))
		level.TilesetUsage = append(level.TilesetUsage, cloneTileUsageSlice(layer.TilesetUsage))
		level.LayerMeta = append(level.LayerMeta, levels.LayerMeta{Physics: layer.Physics, Name: layer.Name})
	}

	return level
}

func (s TileSelection) Normalize() TileSelection {
	result := s
	if result.TileW <= 0 {
		result.TileW = DefaultTileSize
	}
	if result.TileH <= 0 {
		result.TileH = DefaultTileSize
	}
	if result.Index < 0 {
		result.Index = 0
	}
	return result
}

func (s TileSelection) ToTileInfo() *levels.TileInfo {
	normalized := s.Normalize()
	if normalized.Path == "" {
		return nil
	}
	return &levels.TileInfo{
		Path:  normalized.Path,
		Index: normalized.Index,
		TileW: normalized.TileW,
		TileH: normalized.TileH,
	}
}

func InferSelection(doc *LevelDocument, assetNames []string) TileSelection {
	if doc != nil {
		for _, layer := range doc.Layers {
			for _, usage := range layer.TilesetUsage {
				if usage != nil && usage.Path != "" {
					return TileSelection{
						Path:  usage.Path,
						Index: usage.Index,
						TileW: usage.TileW,
						TileH: usage.TileH,
					}.Normalize()
				}
			}
		}
	}
	if len(assetNames) > 0 {
		return TileSelection{Path: assetNames[0], Index: 0, TileW: DefaultTileSize, TileH: DefaultTileSize}
	}
	return TileSelection{TileW: DefaultTileSize, TileH: DefaultTileSize}
}

func tilesetUsageAt(all [][]*levels.TileInfo, index int) []*levels.TileInfo {
	if index < 0 || index >= len(all) {
		return nil
	}
	return all[index]
}

func defaultLayerName(index, total int) string {
	switch {
	case total == 2 && index == 0:
		return "Background"
	case total == 2 && index == 1:
		return "Physics"
	default:
		return fmt.Sprintf("Layer %d", index+1)
	}
}

func cloneTileUsageSlice(usage []*levels.TileInfo) []*levels.TileInfo {
	if usage == nil {
		return nil
	}
	cloned := make([]*levels.TileInfo, len(usage))
	for index, item := range usage {
		cloned[index] = cloneTileInfo(item)
	}
	return cloned
}

func cloneTileInfo(info *levels.TileInfo) *levels.TileInfo {
	if info == nil {
		return nil
	}
	clone := *info
	return &clone
}

func cloneEntities(entities []levels.Entity) []levels.Entity {
	if entities == nil {
		return nil
	}
	cloned := make([]levels.Entity, 0, len(entities))
	for _, entity := range entities {
		copied := entity
		copied.Props = cloneProps(entity.Props)
		cloned = append(cloned, copied)
	}
	return cloned
}

func cloneProps(props map[string]interface{}) map[string]interface{} {
	if props == nil {
		return nil
	}
	encoded, err := json.Marshal(props)
	if err != nil {
		cloned := make(map[string]interface{}, len(props))
		for key, value := range props {
			cloned[key] = value
		}
		return cloned
	}
	var cloned map[string]interface{}
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		fallback := make(map[string]interface{}, len(props))
		for key, value := range props {
			fallback[key] = value
		}
		return fallback
	}
	return cloned
}
