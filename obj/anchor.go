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
	// extension animation state
	Extending   bool
	ExtProgress float64
	ExtDuration float64
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
	// start extension animation toward the exact world coords; create the
	// joint only after the extension completes
	a.Pos = cp.Vector{X: mx, Y: my}
	a.Extending = true
	a.ExtProgress = 0
	a.ExtDuration = 0.15
	// mark as not yet active until joint created
	a.Active = false
	a.Joint = nil
	return true
}

// Update advances extension animation and creates the joint when finished.
func (a *Anchor) Update() {
	if a == nil || !a.Extending || a.Player == nil || a.Player.body == nil || a.Player.CollisionWorld == nil {
		return
	}
	tps := ebiten.ActualTPS()
	if tps <= 0 {
		tps = 60
	}
	dt := 1.0 / tps
	a.ExtProgress += dt
	if a.ExtProgress >= a.ExtDuration {
		// finalize attach: create joint at a.Pos
		anchorWorld := a.Pos
		bpos := a.Player.body.Position()
		dx := bpos.X - anchorWorld.X
		dy := bpos.Y - anchorWorld.Y
		dist := math.Hypot(dx, dy)

		anchorA := a.Player.body.WorldToLocal(anchorWorld)
		anchorB := a.Player.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

		joint := cp.NewSlideJoint(a.Player.body, a.Player.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
		a.Player.CollisionWorld.space.AddConstraint(joint)
		a.Joint = joint
		a.Active = true
		a.Extending = false
		a.ExtProgress = a.ExtDuration
	}
}

// CreateJointAt immediately creates the physics joint at the provided world
// coordinates without playing the extension animation. Returns true on
// success.
func (a *Anchor) CreateJointAt(mx, my float64) bool {
	if a == nil || a.Player == nil || a.Player.CollisionWorld == nil || a.Player.body == nil {
		return false
	}
	tx := int(math.Floor(mx / float64(common.TileSize)))
	ty := int(math.Floor(my / float64(common.TileSize)))
	if !a.Player.CollisionWorld.level.physicsTileAt(tx, ty) {
		return false
	}
	anchorWorld := cp.Vector{X: mx, Y: my}
	bpos := a.Player.body.Position()
	dx := bpos.X - anchorWorld.X
	dy := bpos.Y - anchorWorld.Y
	dist := math.Hypot(dx, dy)

	anchorA := a.Player.body.WorldToLocal(anchorWorld)
	anchorB := a.Player.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

	joint := cp.NewSlideJoint(a.Player.body, a.Player.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
	a.Player.CollisionWorld.space.AddConstraint(joint)
	a.Joint = joint
	a.Active = true
	a.Extending = false
	a.ExtProgress = a.ExtDuration
	a.Pos = anchorWorld
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
	if a == nil || a.Player == nil {
		return
	}
	bpos := a.Player.body.Position()
	px := (bpos.X - camX) * zoom
	py := (bpos.Y - camY) * zoom

	// if extending, draw partial line from player toward a.Pos based on progress
	if a.Extending {
		// t in [0,1]
		t := a.ExtProgress / a.ExtDuration
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}
		tx := (a.Pos.X - camX) * zoom
		ty := (a.Pos.Y - camY) * zoom
		ix := float32(px + (tx-px)*t)
		iy := float32(py + (ty-py)*t)
		vector.StrokeLine(screen, float32(px), float32(py), ix, iy, 3, colornames.Lightgrey, true)
		return
	}

	// otherwise draw if active
	if a.Active {
		vector.StrokeLine(screen, float32(px), float32(py), float32((a.Pos.X-camX)*zoom), float32((a.Pos.Y-camY)*zoom), 3, colornames.Lightgrey, true)
	}
}
