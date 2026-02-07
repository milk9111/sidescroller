package main

// Level represents a game level with layers, entities, and metadata.
type Level struct {
	Name     string    `json:"name"`
	Width    int       `json:"width"`
	Height   int       `json:"height"`
	Layers   []Layer   `json:"layers"`
	Entities []Entity  `json:"entities"`
	Meta     LevelMeta `json:"meta"`
}

// Layer represents a single tile layer in the level.
type Layer struct {
	Name    string     `json:"name"`
	Tiles   [][]Tile   `json:"tiles"`
	Visible bool       `json:"visible"`
	Tint    [4]float32 `json:"tint"`
	Meta    LayerMeta  `json:"meta"`
}

// Tile represents a single tile in a layer.
type Tile struct {
	Tileset string `json:"tileset"`
	Index   int    `json:"index"`
	Physics bool   `json:"physics"`
}

// Entity represents a placed entity in the level.
type Entity struct {
	Type  string                 `json:"type"`
	X     int                    `json:"x"`
	Y     int                    `json:"y"`
	Props map[string]interface{} `json:"props"`
}

// LevelMeta holds metadata for the level.
type LevelMeta struct {
	Background string `json:"background"`
}

// LayerMeta holds metadata for a layer.
type LayerMeta struct {
	Physics bool `json:"physics"`
}
