package levels

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
)

//go:embed *.json
var LevelsFS embed.FS

type Level struct {
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Layers       [][]int       `json:"layers"`
	TilesetUsage [][]*TileInfo `json:"tileset_usage"`
	LayerMeta    []LayerMeta   `json:"layer_meta,omitempty"`
	Entities     []Entity      `json:"entities,omitempty"`
}

type LayerMeta struct {
	Physics bool `json:"physics"`
}

type Entity struct {
	Type  string                 `json:"type"`
	X     int                    `json:"x"`
	Y     int                    `json:"y"`
	Props map[string]interface{} `json:"props,omitempty"`
}

type TileInfo struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	TileW int    `json:"tile_w"`
	TileH int    `json:"tile_h"`
	// Autotile metadata (optional)
	Auto      bool  `json:"auto,omitempty"`
	BaseIndex int   `json:"base_index,omitempty"`
	Mask      uint8 `json:"mask,omitempty"`
}

func LoadLevelFromFS(name string) (*Level, error) {
	data, err := fs.ReadFile(LevelsFS, name)
	if err != nil {
		return nil, fmt.Errorf("read level: %w", err)
	}
	var lvl Level
	if err := json.Unmarshal(data, &lvl); err != nil {
		return nil, fmt.Errorf("unmarshal level: %w", err)
	}
	return &lvl, nil
}
