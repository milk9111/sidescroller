package model

import editormodel "github.com/milk9111/sidescroller/cmd/editor/model"

const DefaultTileSize = editormodel.DefaultTileSize

type TileSelection = editormodel.TileSelection
type Layer = editormodel.Layer
type LevelDocument = editormodel.LevelDocument
type Snapshot = editormodel.Snapshot

var NewLevelDocument = editormodel.NewLevelDocument
var FromRuntimeLevel = editormodel.FromRuntimeLevel
var InferSelection = editormodel.InferSelection
