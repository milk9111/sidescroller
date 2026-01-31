package obj

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"golang.org/x/image/colornames"
)

// Anchor holds swing/attach information for a player.
type Anchor struct {
	Player         *Player
	Camera         *Camera
	CollisionWorld *CollisionWorld

	Active bool
	Pos    cp.Vector
	Joint  *cp.Constraint
	Angle  float64

	// extension animation state
	Extending   bool
	ExtProgress float64
	ExtDuration float64
	// InitialAngle stores the rotation (radians) the claw should keep
	// based on the direction from the player to the attach point at time
	// of attachment.
	InitialAngle float64
}

func NewAnchor() *Anchor {
	return &Anchor{}
}

func (a *Anchor) Init(
	player *Player,
	camera *Camera,
	collisionWorld *CollisionWorld,
) {
	a.Player = player
	a.Camera = camera
	a.CollisionWorld = collisionWorld
}

// Attach attempts to attach to the physics tile under world coords mx,my.
// Returns true if attached.
// Attach attaches at the exact world coordinates provided (mx,my). Returns
// true if the attachment point lies on a physics tile and the joint was
// created successfully.
func (a *Anchor) Attach(mx, my float64) bool {
	txi := int(math.Floor(mx / float64(common.TileSize)))
	tyi := int(math.Floor(my / float64(common.TileSize)))
	if !a.CollisionWorld.level.physicsTileAt(txi, tyi) {
		return false
	}

	tx := (a.Pos.X - a.Camera.PosX) * a.Camera.Zoom()
	ty := (a.Pos.Y - a.Camera.PosY) * a.Camera.Zoom()
	bpos := a.Player.body.Position()
	px := (bpos.X - a.Camera.PosX) * a.Camera.Zoom()
	py := (bpos.Y - a.Camera.PosY) * a.Camera.Zoom()
	ix := float32(px + (tx - px))
	iy := float32(py + (ty - py))

	// compute angle from player (px,py) to tip (ix,iy)
	a.Angle = math.Atan2(float64(iy)-py, float64(ix)-px) + math.Pi/2

	// start extension animation toward the exact world coords; create the
	// joint only after the extension completes
	a.Pos = cp.Vector{X: mx, Y: my}
	// record initial angle from player to attach point so claw keeps this
	if a.Player != nil && a.Player.body != nil {
		bpos := a.Player.body.Position()
		a.InitialAngle = math.Atan2(a.Pos.Y-bpos.Y, a.Pos.X-bpos.X) + math.Pi/2
	}
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
		anchorB := a.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

		joint := cp.NewSlideJoint(a.Player.body, a.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
		a.CollisionWorld.space.AddConstraint(joint)
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
	anchorB := a.CollisionWorld.space.StaticBody.WorldToLocal(anchorWorld)

	joint := cp.NewSlideJoint(a.Player.body, a.CollisionWorld.space.StaticBody, anchorA, anchorB, 0, dist)
	a.CollisionWorld.space.AddConstraint(joint)
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
		a.CollisionWorld.space.RemoveConstraint(a.Joint)
	}
	a.Active = false
	a.Joint = nil
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
		// draw claw at the current extension tip using the stored initial angle
		if assets.Claw != nil {
			w, h := assets.Claw.Size()
			op := &ebiten.DrawImageOptions{}
			scale := 0.5
			// scale, translate so bottom-center is origin, rotate by InitialAngle, then translate to screen position
			op.GeoM.Scale(scale, scale)
			op.GeoM.Translate(-float64(w)*scale/2.0, -float64(h)*scale)
			op.GeoM.Rotate(a.InitialAngle)
			op.GeoM.Translate(float64(ix), float64(iy))
			op.Filter = ebiten.FilterLinear
			screen.DrawImage(assets.Claw, op)
		}
		return
	}

	// otherwise draw if active
	if a.Active {
		ax := float64((a.Pos.X - camX) * zoom)
		ay := float64((a.Pos.Y - camY) * zoom)
		vector.StrokeLine(screen, float32(px), float32(py), float32(ax), float32(ay), 3, colornames.Lightgrey, true)
		// draw claw sprite at anchor point using the stored initial angle
		if assets.Claw != nil {
			w, h := assets.Claw.Size()
			op := &ebiten.DrawImageOptions{}
			scale := 0.5
			op.GeoM.Scale(scale, scale)
			// move origin to bottom-center so bottom-middle of sprite sits at anchor
			op.GeoM.Translate(-float64(w)*scale/2.0, -float64(h)*scale)
			op.GeoM.Rotate(a.InitialAngle)
			op.GeoM.Translate(ax, ay)
			op.Filter = ebiten.FilterLinear
			screen.DrawImage(assets.Claw, op)
		}
	}
}
