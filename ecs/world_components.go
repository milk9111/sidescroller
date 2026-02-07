package ecs

import "github.com/milk9111/sidescroller/ecs/components"

// Transforms returns the transform storage.
func (w *World) Transforms() *SparseSet {
	if w == nil {
		return nil
	}
	if w.transforms == nil {
		w.transforms = &SparseSet{}
	}
	return w.transforms
}

// Sprites returns the sprite storage.
func (w *World) Sprites() *SparseSet {
	if w == nil {
		return nil
	}
	if w.sprites == nil {
		w.sprites = &SparseSet{}
	}
	return w.sprites
}

// Animators returns the animator storage.
func (w *World) Animators() *SparseSet {
	if w == nil {
		return nil
	}
	if w.animators == nil {
		w.animators = &SparseSet{}
	}
	return w.animators
}

// Velocities returns the velocity storage.
func (w *World) Velocities() *SparseSet {
	if w == nil {
		return nil
	}
	if w.velocities == nil {
		w.velocities = &SparseSet{}
	}
	return w.velocities
}

// Accelerations returns the acceleration storage.
func (w *World) Accelerations() *SparseSet {
	if w == nil {
		return nil
	}
	if w.accels == nil {
		w.accels = &SparseSet{}
	}
	return w.accels
}

// Gravities returns the gravity storage.
func (w *World) Gravities() *SparseSet {
	if w == nil {
		return nil
	}
	if w.gravities == nil {
		w.gravities = &SparseSet{}
	}
	return w.gravities
}

// Colliders returns the collider storage.
func (w *World) Colliders() *SparseSet {
	if w == nil {
		return nil
	}
	if w.colliders == nil {
		w.colliders = &SparseSet{}
	}
	return w.colliders
}

// GroundSensors returns the ground sensor storage.
func (w *World) GroundSensors() *SparseSet {
	if w == nil {
		return nil
	}
	if w.grounders == nil {
		w.grounders = &SparseSet{}
	}
	return w.grounders
}

// PhysicsBodies returns the physics body storage.
func (w *World) PhysicsBodies() *SparseSet {
	if w == nil {
		return nil
	}
	if w.physBodies == nil {
		w.physBodies = &SparseSet{}
	}
	return w.physBodies
}

// CollisionStates returns the collision state storage.
func (w *World) CollisionStates() *SparseSet {
	if w == nil {
		return nil
	}
	if w.collStates == nil {
		w.collStates = &SparseSet{}
	}
	return w.collStates
}

// Healths returns the health storage.
func (w *World) Healths() *SparseSet {
	if w == nil {
		return nil
	}
	if w.healths == nil {
		w.healths = &SparseSet{}
	}
	return w.healths
}

// DamageDealers returns the damage dealer storage.
func (w *World) DamageDealers() *SparseSet {
	if w == nil {
		return nil
	}
	if w.dealers == nil {
		w.dealers = &SparseSet{}
	}
	return w.dealers
}

// Hurtboxes returns the hurtbox storage.
func (w *World) Hurtboxes() *SparseSet {
	if w == nil {
		return nil
	}
	if w.hurtboxes == nil {
		w.hurtboxes = &SparseSet{}
	}
	return w.hurtboxes
}

// AIStates returns the AI state storage.
func (w *World) AIStates() *SparseSet {
	if w == nil {
		return nil
	}
	if w.aiStates == nil {
		w.aiStates = &SparseSet{}
	}
	return w.aiStates
}

// Pathfindings returns the pathfinding storage.
func (w *World) Pathfindings() *SparseSet {
	if w == nil {
		return nil
	}
	if w.pathing == nil {
		w.pathing = &SparseSet{}
	}
	return w.pathing
}

// Inputs returns the input state storage.
func (w *World) Inputs() *SparseSet {
	if w == nil {
		return nil
	}
	if w.inputs == nil {
		w.inputs = &SparseSet{}
	}
	return w.inputs
}

// Cameras returns the camera follow storage.
func (w *World) Cameras() *SparseSet {
	if w == nil {
		return nil
	}
	if w.cameras == nil {
		w.cameras = &SparseSet{}
	}
	return w.cameras
}

// CameraStates returns the camera state storage.
func (w *World) CameraStates() *SparseSet {
	if w == nil {
		return nil
	}
	if w.cameraStates == nil {
		w.cameraStates = &SparseSet{}
	}
	return w.cameraStates
}

// PlayerControllers returns the player controller storage.
func (w *World) PlayerControllers() *SparseSet {
	if w == nil {
		return nil
	}
	if w.playerCtrls == nil {
		w.playerCtrls = &SparseSet{}
	}
	return w.playerCtrls
}

// Pickups returns the pickup storage.
func (w *World) Pickups() *SparseSet {
	if w == nil {
		return nil
	}
	if w.pickups == nil {
		w.pickups = &SparseSet{}
	}
	return w.pickups
}

// Bullets returns the bullet storage.
func (w *World) Bullets() *SparseSet {
	if w == nil {
		return nil
	}
	if w.bullets == nil {
		w.bullets = &SparseSet{}
	}
	return w.bullets
}

// SetTransform attaches a transform component.
func (w *World) SetTransform(e Entity, t *components.Transform) {
	if w == nil || t == nil {
		return
	}
	w.Transforms().Set(e.ID, t)
}

// GetTransform returns a transform component.
func (w *World) GetTransform(e Entity) *components.Transform {
	if w == nil {
		return nil
	}
	v := w.Transforms().Get(e.ID)
	if t, ok := v.(*components.Transform); ok {
		return t
	}
	return nil
}

// SetSprite attaches a sprite component.
func (w *World) SetSprite(e Entity, s *components.Sprite) {
	if w == nil || s == nil {
		return
	}
	w.Sprites().Set(e.ID, s)
}

// GetSprite returns a sprite component.
func (w *World) GetSprite(e Entity) *components.Sprite {
	if w == nil {
		return nil
	}
	v := w.Sprites().Get(e.ID)
	if s, ok := v.(*components.Sprite); ok {
		return s
	}
	return nil
}

// SetAnimator attaches an animator component.
func (w *World) SetAnimator(e Entity, a *components.Animator) {
	if w == nil || a == nil {
		return
	}
	w.Animators().Set(e.ID, a)
}

// GetAnimator returns an animator component.
func (w *World) GetAnimator(e Entity) *components.Animator {
	if w == nil {
		return nil
	}
	v := w.Animators().Get(e.ID)
	if a, ok := v.(*components.Animator); ok {
		return a
	}
	return nil
}

// SetVelocity attaches a velocity component.
func (w *World) SetVelocity(e Entity, v *components.Velocity) {
	if w == nil || v == nil {
		return
	}
	w.Velocities().Set(e.ID, v)
}

// GetVelocity returns a velocity component.
func (w *World) GetVelocity(e Entity) *components.Velocity {
	if w == nil {
		return nil
	}
	v := w.Velocities().Get(e.ID)
	if c, ok := v.(*components.Velocity); ok {
		return c
	}
	return nil
}

// SetAcceleration attaches an acceleration component.
func (w *World) SetAcceleration(e Entity, a *components.Acceleration) {
	if w == nil || a == nil {
		return
	}
	w.Accelerations().Set(e.ID, a)
}

// GetAcceleration returns an acceleration component.
func (w *World) GetAcceleration(e Entity) *components.Acceleration {
	if w == nil {
		return nil
	}
	v := w.Accelerations().Get(e.ID)
	if c, ok := v.(*components.Acceleration); ok {
		return c
	}
	return nil
}

// SetGravity attaches a gravity component.
func (w *World) SetGravity(e Entity, g *components.Gravity) {
	if w == nil || g == nil {
		return
	}
	w.Gravities().Set(e.ID, g)
}

// GetGravity returns a gravity component.
func (w *World) GetGravity(e Entity) *components.Gravity {
	if w == nil {
		return nil
	}
	v := w.Gravities().Get(e.ID)
	if c, ok := v.(*components.Gravity); ok {
		return c
	}
	return nil
}

// SetCollider attaches a collider component.
func (w *World) SetCollider(e Entity, c *components.Collider) {
	if w == nil || c == nil {
		return
	}
	w.Colliders().Set(e.ID, c)
}

// GetCollider returns a collider component.
func (w *World) GetCollider(e Entity) *components.Collider {
	if w == nil {
		return nil
	}
	v := w.Colliders().Get(e.ID)
	if c, ok := v.(*components.Collider); ok {
		return c
	}
	return nil
}

// SetGroundSensor attaches a ground sensor component.
func (w *World) SetGroundSensor(e Entity, g *components.GroundSensor) {
	if w == nil || g == nil {
		return
	}
	w.GroundSensors().Set(e.ID, g)
}

// GetGroundSensor returns a ground sensor component.
func (w *World) GetGroundSensor(e Entity) *components.GroundSensor {
	if w == nil {
		return nil
	}
	v := w.GroundSensors().Get(e.ID)
	if c, ok := v.(*components.GroundSensor); ok {
		return c
	}
	return nil
}

// SetPhysicsBody attaches a physics body component.
func (w *World) SetPhysicsBody(e Entity, b *components.PhysicsBody) {
	if w == nil || b == nil {
		return
	}
	w.PhysicsBodies().Set(e.ID, b)
}

// GetPhysicsBody returns a physics body component.
func (w *World) GetPhysicsBody(e Entity) *components.PhysicsBody {
	if w == nil {
		return nil
	}
	v := w.PhysicsBodies().Get(e.ID)
	if c, ok := v.(*components.PhysicsBody); ok {
		return c
	}
	return nil
}

// SetCollisionState attaches a collision state component.
func (w *World) SetCollisionState(e Entity, s *components.CollisionState) {
	if w == nil || s == nil {
		return
	}
	w.CollisionStates().Set(e.ID, s)
}

// GetCollisionState returns a collision state component.
func (w *World) GetCollisionState(e Entity) *components.CollisionState {
	if w == nil {
		return nil
	}
	v := w.CollisionStates().Get(e.ID)
	if c, ok := v.(*components.CollisionState); ok {
		return c
	}
	return nil
}

// SetHealth attaches a health component.
func (w *World) SetHealth(e Entity, h *components.Health) {
	if w == nil || h == nil {
		return
	}
	w.Healths().Set(e.ID, h)
}

// GetHealth returns a health component.
func (w *World) GetHealth(e Entity) *components.Health {
	if w == nil {
		return nil
	}
	v := w.Healths().Get(e.ID)
	if h, ok := v.(*components.Health); ok {
		return h
	}
	return nil
}

// SetDamageDealer attaches a damage dealer component.
func (w *World) SetDamageDealer(e Entity, d *components.DamageDealer) {
	if w == nil || d == nil {
		return
	}
	w.DamageDealers().Set(e.ID, d)
}

// GetDamageDealer returns a damage dealer component.
func (w *World) GetDamageDealer(e Entity) *components.DamageDealer {
	if w == nil {
		return nil
	}
	v := w.DamageDealers().Get(e.ID)
	if d, ok := v.(*components.DamageDealer); ok {
		return d
	}
	return nil
}

// SetHurtbox attaches a hurtbox component.
func (w *World) SetHurtbox(e Entity, h *components.HurtboxSet) {
	if w == nil || h == nil {
		return
	}
	w.Hurtboxes().Set(e.ID, h)
}

// GetHurtbox returns a hurtbox component.
func (w *World) GetHurtbox(e Entity) *components.HurtboxSet {
	if w == nil {
		return nil
	}
	v := w.Hurtboxes().Get(e.ID)
	if h, ok := v.(*components.HurtboxSet); ok {
		return h
	}
	return nil
}

// SetAIState attaches an AI state component.
func (w *World) SetAIState(e Entity, a *components.AIState) {
	if w == nil || a == nil {
		return
	}
	w.AIStates().Set(e.ID, a)
}

// GetAIState returns an AI state component.
func (w *World) GetAIState(e Entity) *components.AIState {
	if w == nil {
		return nil
	}
	v := w.AIStates().Get(e.ID)
	if a, ok := v.(*components.AIState); ok {
		return a
	}
	return nil
}

// SetPathfinding attaches a pathfinding component.
func (w *World) SetPathfinding(e Entity, p *components.Pathfinding) {
	if w == nil || p == nil {
		return
	}
	w.Pathfindings().Set(e.ID, p)
}

// GetPathfinding returns a pathfinding component.
func (w *World) GetPathfinding(e Entity) *components.Pathfinding {
	if w == nil {
		return nil
	}
	v := w.Pathfindings().Get(e.ID)
	if p, ok := v.(*components.Pathfinding); ok {
		return p
	}
	return nil
}

// SetInput attaches an input state component.
func (w *World) SetInput(e Entity, i *components.InputState) {
	if w == nil || i == nil {
		return
	}
	w.Inputs().Set(e.ID, i)
}

// GetInput returns an input state component.
func (w *World) GetInput(e Entity) *components.InputState {
	if w == nil {
		return nil
	}
	v := w.Inputs().Get(e.ID)
	if i, ok := v.(*components.InputState); ok {
		return i
	}
	return nil
}

// SetCameraFollow attaches a camera follow component.
func (w *World) SetCameraFollow(e Entity, c *components.CameraFollow) {
	if w == nil || c == nil {
		return
	}
	w.Cameras().Set(e.ID, c)
}

// GetCameraFollow returns a camera follow component.
func (w *World) GetCameraFollow(e Entity) *components.CameraFollow {
	if w == nil {
		return nil
	}
	v := w.Cameras().Get(e.ID)
	if c, ok := v.(*components.CameraFollow); ok {
		return c
	}
	return nil
}

// SetCameraState attaches a camera state component.
func (w *World) SetCameraState(e Entity, c *components.CameraState) {
	if w == nil || c == nil {
		return
	}
	w.CameraStates().Set(e.ID, c)
}

// GetCameraState returns a camera state component.
func (w *World) GetCameraState(e Entity) *components.CameraState {
	if w == nil {
		return nil
	}
	v := w.CameraStates().Get(e.ID)
	if c, ok := v.(*components.CameraState); ok {
		return c
	}
	return nil
}

// SetPlayerController attaches a player controller component.
func (w *World) SetPlayerController(e Entity, p *components.PlayerController) {
	if w == nil || p == nil {
		return
	}
	w.PlayerControllers().Set(e.ID, p)
}

// GetPlayerController returns a player controller component.
func (w *World) GetPlayerController(e Entity) *components.PlayerController {
	if w == nil {
		return nil
	}
	v := w.PlayerControllers().Get(e.ID)
	if p, ok := v.(*components.PlayerController); ok {
		return p
	}
	return nil
}

// SetPickup attaches a pickup component.
func (w *World) SetPickup(e Entity, p *components.Pickup) {
	if w == nil || p == nil {
		return
	}
	w.Pickups().Set(e.ID, p)
}

// GetPickup returns a pickup component.
func (w *World) GetPickup(e Entity) *components.Pickup {
	if w == nil {
		return nil
	}
	v := w.Pickups().Get(e.ID)
	if p, ok := v.(*components.Pickup); ok {
		return p
	}
	return nil
}

// SetBullet attaches a bullet component.
func (w *World) SetBullet(e Entity, b *components.Bullet) {
	if w == nil || b == nil {
		return
	}
	w.Bullets().Set(e.ID, b)
}

// GetBullet returns a bullet component.
func (w *World) GetBullet(e Entity) *components.Bullet {
	if w == nil {
		return nil
	}
	v := w.Bullets().Get(e.ID)
	if b, ok := v.(*components.Bullet); ok {
		return b
	}
	return nil
}
