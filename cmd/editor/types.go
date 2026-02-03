package main

type Level struct {
	Width     int         `json:"width"`
	Height    int         `json:"height"`
	Layers    [][]int     `json:"layers,omitempty"` // optional layers, each row-major
	LayerMeta []LayerMeta `json:"layer_meta,omitempty"`
	SpawnX    int         `json:"spawn_x,omitempty"`
	SpawnY    int         `json:"spawn_y,omitempty"`
	// optional background image entries for parallax layers
	Backgrounds []BackgroundEntry `json:"backgrounds,omitempty"`
	// TilesetUsage stores per-layer, per-cell tileset metadata when a tileset tile is used.
	TilesetUsage [][][]*TilesetEntry `json:"tileset_usage,omitempty"`

	// Transitions are rectangular zones saved in the level JSON that point
	// to another file (Target). Coordinates and size are in tile units.
	Transitions []Transition `json:"transitions,omitempty"`
	// Entities are placed entity instances saved in the level file.
	Entities []PlacedEntity `json:"entities,omitempty"`
}

// Transition defines a rectangular transition zone in tile coordinates.
type Transition struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	ID        string `json:"id,omitempty"`
	Target    string `json:"target"`
	LinkID    string `json:"link_id,omitempty"`
	Direction string `json:"direction,omitempty"`
}

// TilesetEntry records which tileset file and tile index plus tile size used for a cell.
type TilesetEntry struct {
	Path  string `json:"path"`
	Index int    `json:"index"`
	TileW int    `json:"tile_w"`
	TileH int    `json:"tile_h"`
}

type LayerMeta struct {
	HasPhysics bool    `json:"has_physics"`
	Color      string  `json:"color"`
	Name       string  `json:"name,omitempty"`
	Parallax   float64 `json:"parallax,omitempty"`
}

// PlacedEntity represents an entity instance placed in a level.
type PlacedEntity struct {
	Name   string `json:"name"`
	Sprite string `json:"sprite"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
}

// BackgroundEntry stores a background image reference and optional parallax factor.
type BackgroundEntry struct {
	Path     string  `json:"path"`
	Parallax float64 `json:"parallax,omitempty"`
}

const (
	// Increase base width to accomodate tileset panel to the right
	baseWidthEditor  = 40*32 + 220 // 1280 + 220 = 1500
	baseHeightEditor = 23 * 32     // 736
)

// ControlsText renders the on-canvas controls/help text.
type ControlsText struct {
	X int
	Y int
}
