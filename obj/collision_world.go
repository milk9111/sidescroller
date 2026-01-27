package obj

import (
	"github.com/milk9111/sidescroller/common"
)

type CollisionWorld struct {
	level *Level
}

func NewCollisionWorld(level *Level) *CollisionWorld {
	return &CollisionWorld{level: level}
}

// MoveX moves rect horizontally by dx and resolves collisions.
// It returns the resolved rect, whether a collision occurred, and the tile value collided (0 if none).
func (cw *CollisionWorld) MoveX(rect common.Rect, dx float32) (common.Rect, bool, int) {
	// No horizontal movement -> don't resolve horizontal collisions.
	if dx == 0 {
		return rect, false, 0
	}

	// predict horizontal movement and check only immediate left/right neighbors
	moved := rect
	moved.X += dx
	var hit bool
	var collidedTile common.Rect
	if dx > 0 {
		// moving right: check the immediate right column
		minX := float32(1e9)
		for _, tile := range cw.level.QueryHorizontal(rect) {
			if moved.Intersects(&tile) {
				if tile.X < minX {
					minX = tile.X
					hit = true
					collidedTile = tile
				}
			}
		}
		if hit {
			rect.X = minX - rect.Width
			// determine tile value at collided tile
			tx := int(collidedTile.X) / common.TileSize
			ty := int(collidedTile.Y) / common.TileSize
			tileVal := cw.level.TileValueAt(tx, ty)
			return rect, true, tileVal
		}
	} else {
		// moving left: check the immediate left column
		maxRight := float32(-1e9)
		for _, tile := range cw.level.QueryHorizontal(rect) {
			if moved.Intersects(&tile) {
				right := tile.X + tile.Width
				if right > maxRight {
					maxRight = right
					hit = true
					collidedTile = tile
				}
			}
		}
		if hit {
			rect.X = maxRight
			tx := int(collidedTile.X) / common.TileSize
			ty := int(collidedTile.Y) / common.TileSize
			tileVal := cw.level.TileValueAt(tx, ty)
			return rect, true, tileVal
		}
	}
	return rect, false, 0
}

// MoveY moves rect vertically by dy and resolves collisions.
// It returns the resolved rect, whether a collision occurred, and the tile value collided (0 if none).
func (cw *CollisionWorld) MoveY(rect common.Rect, dy float32) (common.Rect, bool, int) {
	// No vertical movement -> don't resolve vertical collisions.
	if dy == 0 {
		return rect, false, 0
	}

	// predict vertical movement and check only immediate top/bottom neighbors
	moved := rect
	moved.Y += dy
	var hit bool
	var collidedTile common.Rect
	if dy > 0 {
		// moving down: check the immediate bottom row
		minY := float32(1e9)
		for _, tile := range cw.level.QueryVertical(rect) {
			if moved.Intersects(&tile) {
				if tile.Y < minY {
					minY = tile.Y
					hit = true
					collidedTile = tile
				}
			}
		}
		if hit {
			rect.Y = minY - rect.Height
			tx := int(collidedTile.X) / common.TileSize
			ty := int(collidedTile.Y) / common.TileSize
			tileVal := cw.level.TileValueAt(tx, ty)
			return rect, true, tileVal
		}
	} else {
		// moving up: check the immediate top row
		maxBottom := float32(-1e9)
		for _, tile := range cw.level.QueryVertical(rect) {
			if moved.Intersects(&tile) {
				bottom := tile.Y + tile.Height
				if bottom > maxBottom {
					maxBottom = bottom
					hit = true
					collidedTile = tile
				}
			}
		}
		if hit {
			rect.Y = maxBottom
			tx := int(collidedTile.X) / common.TileSize
			ty := int(collidedTile.Y) / common.TileSize
			tileVal := cw.level.TileValueAt(tx, ty)
			return rect, true, tileVal
		}
	}
	return rect, false, 0
}

// IsGrounded returns true when r is exactly on top of a non-zero tile.
// It returns false when not touching anything below or when touching
// the bottom of a tile (i.e. not standing on top).
func (cw *CollisionWorld) IsGrounded(r common.Rect) bool {
	if cw == nil || cw.level == nil {
		return false
	}
	return cw.level.IsGrounded(r)
}

// IsTouchingWall returns true when r is exactly touching a non-zero tile
// on the left or right side.
func (cw *CollisionWorld) IsTouchingWall(r common.Rect) wallSide {
	if cw == nil || cw.level == nil {
		return WALL_NONE
	}
	return cw.level.IsTouchingWall(r)
}
