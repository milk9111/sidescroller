package system

import (
	"fmt"
	"strings"

	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/levels"
	"github.com/milk9111/sidescroller/obj"
)

// World owns level loading, transition handling, and spawn logic.
type World struct {
	Level          *obj.Level
	CollisionWorld *obj.CollisionWorld
	Pickups        []*obj.Pickup
	Enemies        []*obj.Enemy
	FlyingEnemies  []*obj.FlyingEnemy
}

// NewWorld creates a new world and loads the requested level.
func NewWorld(levelPath string) (*World, error) {
	w := &World{}
	if err := w.Load(levelPath); err != nil {
		return nil, err
	}
	return w, nil
}

// Load loads a level into the world and resets collision/spawn data.
func (w *World) Load(levelPath string) error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}
	lvl, err := loadLevel(levelPath)
	if err != nil {
		return err
	}
	w.Level = lvl
	w.CollisionWorld = obj.NewCollisionWorld(w.Level)
	w.Pickups = nil
	w.Enemies = nil
	w.FlyingEnemies = nil
	return nil
}

// SpawnEntities spawns pickups and enemies from level entities.
func (w *World) SpawnEntities(player *obj.Player) {
	if w == nil || w.Level == nil || len(w.Level.Entities) == 0 {
		return
	}
	remaining := w.Level.Entities
	w.Pickups, remaining = w.spawnPickupsFromEntities(remaining, player)
	w.FlyingEnemies, remaining = w.spawnFlyingEnemiesFromEntities(remaining)
	w.Enemies, remaining = w.spawnEnemiesFromEntities(remaining)
	w.Level.Entities = remaining
}

// HandleTransition loads the target level and rebuilds world state and player.
func (w *World) HandleTransition(target, linkID, direction string, input *obj.Input, camera *obj.Camera, player *obj.Player) (*obj.Player, *obj.Anchor, error) {
	if w == nil {
		return nil, nil, fmt.Errorf("world is nil")
	}
	if err := w.Load(target); err != nil {
		return nil, nil, err
	}
	if w.Level == nil {
		return nil, nil, fmt.Errorf("failed to load transition target %s", target)
	}

	spawnX, spawnY := w.Level.GetSpawnPosition()
	var targetTr *obj.TransitionData
	if linkID != "" {
		for i := range w.Level.Transitions {
			t2 := &w.Level.Transitions[i]
			if t2.ID == linkID {
				spawnX = float32(t2.X * common.TileSize)
				spawnY = float32(t2.Y * common.TileSize)
				targetTr = t2
				break
			}
		}
	}

	dir := "left"
	d := strings.ToLower(direction)
	switch d {
	case "up", "down", "left", "right":
		dir = d
	default:
		dir = "left"
	}

	if targetTr != nil {
		if dir == "up" || dir == "down" {
			centerX := float32(targetTr.X*common.TileSize) + float32(targetTr.W*common.TileSize)/2.0
			centerY := float32(targetTr.Y*common.TileSize) + float32(targetTr.H*common.TileSize)/2.0
			spawnX = centerX - 8.0
			spawnY = centerY - 20.0
		} else if dir == "left" || dir == "right" {
			centerX := float32(targetTr.X*common.TileSize) + float32(targetTr.W*common.TileSize)/2.0
			bottom := float32((targetTr.Y + targetTr.H) * common.TileSize)
			spawnX = centerX - 8.0
			spawnY = bottom - 40.0
		}
	}

	anchor := obj.NewAnchor()
	newPlayer := obj.NewPlayer(
		spawnX,
		spawnY,
		input,
		w.CollisionWorld,
		anchor,
		player.IsFacingRight(),
		player.DoubleJumpEnabled,
		player.WallGrabEnabled,
		player.SwingEnabled,
		player.DashEnabled,
	)
	if input != nil {
		input.Player = newPlayer
	}
	anchor.Init(newPlayer, camera, w.CollisionWorld)

	if strings.ToLower(direction) == "up" {
		newPlayer.ApplyTransitionJumpImpulse()
	}

	if camera != nil && w.Level != nil {
		levelW := w.Level.Width * common.TileSize
		levelH := w.Level.Height * common.TileSize
		camera.SetWorldBounds(levelW, levelH)
		cx := float64(newPlayer.X + float32(newPlayer.Width)/2.0)
		cy := float64(newPlayer.Y + float32(newPlayer.Height)/2.0)
		camera.SnapTo(cx, cy)
	}

	w.SpawnEntities(newPlayer)
	return newPlayer, anchor, nil
}

// loadLevel tries to load a level from embedded assets first, then from disk.
func loadLevel(levelPath string) (*obj.Level, error) {
	if levelPath == "" {
		return nil, fmt.Errorf("level path is empty")
	}
	if l, err := obj.LoadLevelFromFS(levels.LevelsFS, levelPath); err == nil {
		return l, nil
	}
	if l, err := obj.LoadLevel(levelPath); err == nil {
		return l, nil
	}
	return nil, fmt.Errorf("failed to load level %s", levelPath)
}
