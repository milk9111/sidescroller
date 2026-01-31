package obj

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/common"
	"golang.org/x/image/colornames"
)

// Anchor holds swing/attach information for a player.
type Anchor struct {
	Player *Player
	Active bool
	Pos    cp.Vector
	Joint  *cp.Constraint
}

func NewAnchor(p *Player) *Anchor {
	return &Anchor{Player: p}
}

// Attach attempts to attach to the physics tile under world coords mx,my.
// Returns true if attached.
// Attach attaches at the exact world coordinates provided (mx,my). Returns
// true if the attachment point lies on a physics tile and the joint was
// created successfully.
func (a *Anchor) Attach(mx, my float64) bool {
	if a == nil || a.Player == nil || a.Player.CollisionWorld == nil || a.Player.CollisionWorld.level == nil || a.Player.body == nil {
		return false
	}
	tx := int(math.Floor(mx / float64(common.TileSize)))
	ty := int(math.Floor(my / float64(common.TileSize)))
	if !a.Player.CollisionWorld.level.physicsTileAt(tx, ty) {
		return false
	}
	// use the provided world coords directly so the anchor is placed at the
	// collision point along the aiming ray instead of the tile center
	anchorWorld := cp.Vector{X: mx, Y: my}

	bpos := a.Player.body.Position()
	dx := bpos.X - anchorWorld.X
	dy := bpos.Y - anchorWorld.Y
	dist := math.Hypot(dx, dy)

	anchorA := a.Player.body.WorldToLocal(anchorWorld)
	anchorB := a.Player.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

	joint := cp.NewSlideJoint(a.Player.body, a.Player.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
	a.Player.CollisionWorld.space.AddConstraint(joint)
	a.Active = true
	a.Pos = anchorWorld
	a.Joint = joint
	return true
}

func (a *Anchor) Detach() {
	if a == nil || a.Player == nil || a.Player.CollisionWorld == nil || !a.Active {
		return
	}
	if a.Joint != nil {
		a.Player.CollisionWorld.space.RemoveConstraint(a.Joint)
	}
	a.Active = false
	a.Joint = nil
	if a.Player.body != nil {
		a.Player.body.SetAngle(0)
		a.Player.body.SetAngularVelocity(0)
	}
}

// Draw draws the anchor line if active.
func (a *Anchor) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if a == nil || !a.Active || a.Player == nil {
		return
	}
	bpos := a.Player.body.Position()
	cx := (bpos.X - camX) * zoom
	cy := (bpos.Y - camY) * zoom
	vector.StrokeLine(screen, float32(cx), float32(cy), float32((a.Pos.X-camX)*zoom), float32((a.Pos.Y-camY)*zoom), 3, colornames.Lightgrey, true)
}
