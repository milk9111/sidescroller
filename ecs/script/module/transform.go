package module

import (
	"fmt"
	"math"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func spriteVisualCenterOffset(world *ecs.World, target ecs.Entity, tf *component.Transform) (float64, float64, bool) {
	if tf == nil {
		return 0, 0, false
	}

	sprite, ok := ecs.Get(world, target, component.SpriteComponent.Kind())
	if !ok || sprite == nil || sprite.Image == nil {
		return 0, 0, false
	}

	width := sprite.Image.Bounds().Dx()
	height := sprite.Image.Bounds().Dy()
	if sprite.UseSource {
		width = sprite.Source.Dx()
		height = sprite.Source.Dy()
	}
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}

	scaleX := tf.ScaleX
	if scaleX == 0 {
		scaleX = 1
	}
	scaleY := tf.ScaleY
	if scaleY == 0 {
		scaleY = 1
	}

	visualScaleX := scaleX
	if sprite.FacingLeft {
		visualScaleX = -visualScaleX
	}

	return (float64(width)/2 - sprite.OriginX) * visualScaleX, (float64(height)/2 - sprite.OriginY) * scaleY, true
}

func rotateLocalOffset(x, y, angle float64) (float64, float64) {
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)
	return x*cosA - y*sinA, x*sinA + y*cosA
}

func scriptRotationRadians(world *ecs.World, target ecs.Entity, tf *component.Transform) float64 {
	if body, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind()); ok && body != nil && body.Body != nil {
		return normalizedScriptRotation(body.Body.Angle())
	}

	if tf == nil {
		return 0
	}

	if tf.Parent != 0 || tf.WorldRotation != 0 {
		return normalizedScriptRotation(tf.WorldRotation)
	}

	return normalizedScriptRotation(tf.Rotation)
}

func normalizedScriptRotation(rotation float64) float64 {
	rotation = math.Mod(rotation, 2*math.Pi)
	if rotation < 0 {
		rotation += 2 * math.Pi
	}
	return rotation
}

func snappedDirectionComponent(value float64) float64 {
	const epsilon = 1e-9
	switch {
	case math.Abs(value) <= epsilon:
		return 0
	case math.Abs(value-1) <= epsilon:
		return 1
	case math.Abs(value+1) <= epsilon:
		return -1
	default:
		return value
	}
}

func scriptForwardVector(rotation float64) (float64, float64) {
	return snappedDirectionComponent(math.Cos(rotation)), snappedDirectionComponent(math.Sin(rotation))
}

func scriptDownVector(rotation float64) (float64, float64) {
	return snappedDirectionComponent(-math.Sin(rotation)), snappedDirectionComponent(math.Cos(rotation))
}

func TransformModule() Module {
	return Module{
		Name: "transform",
		Build: func(world *ecs.World, _ map[string]ecs.Entity, _ ecs.Entity, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}
			// sig: position() -> (float, float)
			// doc: Returns the current position as an [x, y] array of floats.
			// sig: position() -> map
			// doc: Returns a map with `x` and `y` numeric fields for the entity's position.
			values["position"] = &tengo.UserFunction{Name: "position", Value: func(args ...tengo.Object) (tengo.Object, error) {
				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, fmt.Errorf("entity does not have a transform component")
				}

				return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: tf.X}, &tengo.Float{Value: tf.Y}}}, nil
			}}
			// sig: set_position(x float, y float) -> bool
			// doc: Sets the entity transform position to (x, y).
			// sig: set_position(x float, y float) -> bool
			// doc: Set the entity's position to the given x,y coordinates.
			values["set_position"] = &tengo.UserFunction{Name: "set_position", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 2 {
					return tengo.FalseValue, fmt.Errorf("set_position requires 2 arguments: x and y")
				}

				x := objectAsFloat(args[0])
				y := objectAsFloat(args[1])

				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return tengo.FalseValue, fmt.Errorf("entity does not have a transform component")
				}

				tf.X = x
				tf.Y = y

				if tf.ScaleX == 0 {
					tf.ScaleX = 1
				}

				if tf.ScaleY == 0 {
					tf.ScaleY = 1
				}

				return tengo.TrueValue, nil
			}}

			values["rotate"] = &tengo.UserFunction{Name: "rotate", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("rotate requires 1 argument: angle in degrees")
				}

				angle := objectAsFloat(args[0]) * math.Pi / 180

				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return tengo.FalseValue, fmt.Errorf("entity does not have a transform component")
				}

				preserveVisualCenter := true
				oldCenterX := 0.0
				oldCenterY := 0.0
				if body, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind()); ok && body != nil && body.Body != nil {
					preserveVisualCenter = false
				}
				if preserveVisualCenter {
					if offsetX, offsetY, ok := spriteVisualCenterOffset(world, target, tf); ok {
						rotatedX, rotatedY := rotateLocalOffset(offsetX, offsetY, tf.Rotation)
						oldCenterX = tf.X + rotatedX
						oldCenterY = tf.Y + rotatedY
					} else {
						preserveVisualCenter = false
					}
				}

				tf.Rotation = normalizedScriptRotation(tf.Rotation + angle)

				if preserveVisualCenter {
					if offsetX, offsetY, ok := spriteVisualCenterOffset(world, target, tf); ok {
						rotatedX, rotatedY := rotateLocalOffset(offsetX, offsetY, tf.Rotation)
						tf.X = oldCenterX - rotatedX
						tf.Y = oldCenterY - rotatedY
					}
				}

				if body, ok := ecs.Get(world, target, component.PhysicsBodyComponent.Kind()); ok && body != nil && body.Body != nil {
					body.Body.SetAngle(tf.Rotation)
					body.Body.SetAngularVelocity(0)
				}

				return tengo.TrueValue, nil
			}}

			values["down_vector"] = &tengo.UserFunction{Name: "down_vector", Value: func(args ...tengo.Object) (tengo.Object, error) {
				tf, ok := ecs.Get(world, target, component.TransformComponent.Kind())
				if !ok || tf == nil {
					return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: 0}, &tengo.Float{Value: 0}}}, fmt.Errorf("entity does not have a transform component")
				}

				rad := scriptRotationRadians(world, target, tf)
				downX, downY := scriptDownVector(rad)

				return &tengo.Array{Value: []tengo.Object{&tengo.Float{Value: downX}, &tengo.Float{Value: downY}}}, nil
			}}

			return values
		},
	}
}
