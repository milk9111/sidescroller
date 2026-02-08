package system

import (
	"math"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	defaultPhysicsStep = 1.0 / 60.0
	defaultGravityY    = 1400.0
	defaultFriction    = 0.9
)

type PhysicsSystem struct {
	space     *cp.Space
	world     *ecs.World
	hasBounds bool
}

func NewPhysicsSystem() *PhysicsSystem {
	return &PhysicsSystem{}
}

func (p *PhysicsSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}
	if p.space == nil || p.world != w {
		p.reset(w)
	}

	p.ensureBounds(w)
	p.ensureBodies(w)
	p.space.Step(defaultPhysicsStep)
	p.applyTransforms(w)
}

func (p *PhysicsSystem) Space() *cp.Space {
	if p == nil {
		return nil
	}
	return p.space
}

func (p *PhysicsSystem) reset(w *ecs.World) {
	p.world = w
	p.space = cp.NewSpace()
	p.space.SetGravity(cp.Vector{X: 0, Y: defaultGravityY})
	p.hasBounds = false
}

func (p *PhysicsSystem) ensureBounds(w *ecs.World) {
	if p.hasBounds || p.space == nil {
		return
	}
	boundsEntity, ok := w.First(component.LevelBoundsComponent.Kind())
	if !ok {
		return
	}
	bounds, ok := ecs.Get(w, boundsEntity, component.LevelBoundsComponent)
	if !ok || bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	staticBody := p.space.StaticBody
	segments := []*cp.Shape{
		cp.NewSegment(staticBody, cp.Vector{X: 0, Y: 0}, cp.Vector{X: bounds.Width, Y: 0}, 0),
		cp.NewSegment(staticBody, cp.Vector{X: 0, Y: bounds.Height}, cp.Vector{X: bounds.Width, Y: bounds.Height}, 0),
		cp.NewSegment(staticBody, cp.Vector{X: 0, Y: 0}, cp.Vector{X: 0, Y: bounds.Height}, 0),
		cp.NewSegment(staticBody, cp.Vector{X: bounds.Width, Y: 0}, cp.Vector{X: bounds.Width, Y: bounds.Height}, 0),
	}

	for _, seg := range segments {
		seg.SetFriction(0)
		p.space.AddShape(seg)
	}

	p.hasBounds = true
}

func (p *PhysicsSystem) ensureBodies(w *ecs.World) {
	if p.space == nil {
		return
	}

	for _, e := range w.Query(component.TransformComponent.Kind(), component.PhysicsBodyComponent.Kind()) {
		transform, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}
		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok {
			continue
		}

		if bodyComp.Width == 0 || bodyComp.Height == 0 {
			if sprite, ok := ecs.Get(w, e, component.SpriteComponent); ok && sprite.Image != nil {
				bounds := sprite.Image.Bounds()
				if bodyComp.Width == 0 {
					bodyComp.Width = float64(bounds.Dx())
				}
				if bodyComp.Height == 0 {
					bodyComp.Height = float64(bounds.Dy())
				}
			}
			if bodyComp.Width == 0 {
				bodyComp.Width = 1
			}
			if bodyComp.Height == 0 {
				bodyComp.Height = 1
			}
		}

		if bodyComp.Body == nil || bodyComp.Shape == nil {
			p.createBody(&bodyComp, transform)
			if err := ecs.Add(w, e, component.PhysicsBodyComponent, bodyComp); err != nil {
				panic("physics system: update body component: " + err.Error())
			}
		}
	}
}

func (p *PhysicsSystem) createBody(bodyComp *component.PhysicsBody, transform component.Transform) {
	if p.space == nil {
		return
	}

	mass := bodyComp.Mass
	if mass <= 0 {
		mass = 1
	}
	width := math.Max(1, bodyComp.Width)
	height := math.Max(1, bodyComp.Height)
	moment := mass * (width*width + height*height) / 12.0

	var body *cp.Body
	if bodyComp.Static {
		body = cp.NewStaticBody()
	} else {
		body = cp.NewBody(mass, moment)
	}

	posX := transform.X
	posY := transform.Y
	if bodyComp.AlignTopLeft {
		posX += width / 2
		posY += height / 2
	}
	// apply optional collider offsets (already in entity-space units)
	posX += bodyComp.OffsetX
	posY += bodyComp.OffsetY
	body.SetPosition(cp.Vector{X: posX, Y: posY})
	shape := cp.NewBox(body, width, height, bodyComp.Radius)

	friction := bodyComp.Friction

	shape.SetFriction(friction)
	if bodyComp.Elasticity > 0 {
		shape.SetElasticity(bodyComp.Elasticity)
	}

	p.space.AddBody(body)
	p.space.AddShape(shape)

	bodyComp.Body = body
	bodyComp.Shape = shape
}

func (p *PhysicsSystem) applyTransforms(w *ecs.World) {
	if p.space == nil {
		return
	}
	for _, e := range w.Query(component.TransformComponent.Kind(), component.PhysicsBodyComponent.Kind()) {
		transform, ok := ecs.Get(w, e, component.TransformComponent)
		if !ok {
			continue
		}
		bodyComp, ok := ecs.Get(w, e, component.PhysicsBodyComponent)
		if !ok || bodyComp.Body == nil {
			continue
		}
		pos := bodyComp.Body.Position()
		if bodyComp.AlignTopLeft {
			transform.X = pos.X - (bodyComp.Width / 2) - bodyComp.OffsetX
			transform.Y = pos.Y - (bodyComp.Height / 2) - bodyComp.OffsetY
		} else {
			transform.X = pos.X - bodyComp.OffsetX
			transform.Y = pos.Y - bodyComp.OffsetY
		}
		transform.Rotation = bodyComp.Body.Angle()
		if err := ecs.Add(w, e, component.TransformComponent, transform); err != nil {
			panic("physics system: update transform: " + err.Error())
		}
	}
}
