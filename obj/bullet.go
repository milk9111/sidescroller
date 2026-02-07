//go:build legacy
// +build legacy

package obj

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/assets"
	"github.com/milk9111/sidescroller/common"
	"github.com/milk9111/sidescroller/component"
)

const bulletSpriteSize = 32

const (
	bulletDamageAmount   = 1
	bulletKnockbackX     = 1.5
	bulletKnockbackY     = -0.5
	bulletIFrameFrames   = 10
	bulletCooldownFrames = 10
)

var (
	bulletPool sync.Pool

	bulletSprite      *ebiten.Image
	bulletPlaceholder *ebiten.Image
	bulletAssetsOnce  sync.Once

	activeBullets []*Bullet
	bulletNextID  int
)

// Bullet represents a simple projectile with its own update and draw behavior.
type Bullet struct {
	ID        int
	OwnerID   int
	X, Y      float32
	VelocityX float32
	VelocityY float32
	Rotation  float64
	Active    bool

	Width  float32
	Height float32

	LifeFrames int
	age        int

	faction       component.Faction
	combatEmitter component.CombatEventEmitter
	hitboxes      []component.Hitbox

	img         *ebiten.Image
	placeholder *ebiten.Image
}

// NewBullet creates a bullet at world pixel (x,y) with the given velocity.
// Prefer SpawnBullet to reuse pooled instances.
func NewBullet(x, y, vx, vy float32) *Bullet {
	bulletAssetsOnce.Do(initBulletAssets)
	b := &Bullet{}
	b.Reset(x, y, vx, vy, 0, 0)
	return b
}

// SpawnBullet pulls a bullet from the pool and tracks it for updates/draws.
func SpawnBullet(x, y, vx, vy float32, rotation float64, ownerID int) *Bullet {
	bulletAssetsOnce.Do(initBulletAssets)
	b := getBulletFromPool()
	b.Reset(x, y, vx, vy, rotation, ownerID)
	activeBullets = append(activeBullets, b)
	return b
}

// UpdateBullets advances all active bullets and releases inactive ones.
func UpdateBullets(player *Player, collisionWorld *CollisionWorld) {
	if len(activeBullets) == 0 {
		return
	}
	writeIdx := 0
	for _, b := range activeBullets {
		if b == nil {
			continue
		}
		b.Update(player, collisionWorld)
		if !b.Active {
			releaseBulletToPool(b)
			continue
		}
		activeBullets[writeIdx] = b
		writeIdx++
	}
	activeBullets = activeBullets[:writeIdx]
}

// DrawBullets renders all active bullets.
func DrawBullets(screen *ebiten.Image, camX, camY, zoom float64) {
	if len(activeBullets) == 0 {
		return
	}
	for _, b := range activeBullets {
		if b != nil {
			b.Draw(screen, camX, camY, zoom)
		}
	}
}

func initBulletAssets() {
	if im, err := assets.LoadImage("flying_enemy_bullet.png"); err == nil {
		bulletSprite = im
	}
	if bulletSprite == nil {
		img := ebiten.NewImage(bulletSpriteSize, bulletSpriteSize)
		img.Fill(color.RGBA{R: 0xff, G: 0x00, B: 0xff, A: 0xff})
		bulletPlaceholder = img
	}
}

func getBulletFromPool() *Bullet {
	if bulletPool.New == nil {
		bulletPool.New = func() any { return &Bullet{} }
	}
	return bulletPool.Get().(*Bullet)
}

func releaseBulletToPool(b *Bullet) {
	if b == nil {
		return
	}
	b.Active = false
	b.VelocityX = 0
	b.VelocityY = 0
	b.Rotation = 0
	b.age = 0
	b.LifeFrames = 0
	b.hitboxes = nil
	bulletPool.Put(b)
}

// Reset reinitializes a bullet instance for reuse.
func (b *Bullet) Reset(x, y, vx, vy float32, rotation float64, ownerID int) {
	if b == nil {
		return
	}
	bulletNextID++
	b.ID = bulletNextID
	b.OwnerID = ownerID
	b.X = x
	b.Y = y
	b.VelocityX = vx
	b.VelocityY = vy
	b.Rotation = rotation
	b.Active = true
	b.Width = bulletSpriteSize
	b.Height = bulletSpriteSize
	b.LifeFrames = 0
	b.age = 0
	b.faction = component.FactionEnemy
	b.hitboxes = nil
	b.img = bulletSprite
	b.placeholder = bulletPlaceholder
}

// Update advances the bullet position and handles lifetime expiry.
func (b *Bullet) Update(player *Player, collisionWorld *CollisionWorld) {
	if b == nil || !b.Active {
		return
	}

	b.X += b.VelocityX
	b.Y += b.VelocityY

	if b.collidesWithPhysics(collisionWorld) {
		b.Active = false
		return
	}

	if player != nil {
		b.updateHitbox()
		if b.overlapsPlayer(player) {
			if player.Health() != nil {
				resolver := component.NewCombatResolver()
				resolver.Resolve(b, player, player.Health())
				if len(resolver.Recent) > 0 {
					component.AddRecentHighlights(resolver.Recent)
				}
			}
			b.Active = false
			return
		}
	}

	if b.LifeFrames > 0 {
		b.age++
		if b.age >= b.LifeFrames {
			b.Active = false
		}
	}
}

// Draw renders the bullet with camera transform applied.
func (b *Bullet) Draw(screen *ebiten.Image, camX, camY, zoom float64) {
	if b == nil || !b.Active || screen == nil {
		return
	}
	if zoom <= 0 {
		zoom = 1
	}

	img := b.img
	if img == nil {
		img = b.placeholder
	}
	if img == nil {
		return
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	if w <= 0 || h <= 0 {
		return
	}

	op := &ebiten.DrawImageOptions{}
	scaleX := float64(b.Width) / float64(w)
	scaleY := float64(b.Height) / float64(h)
	op.GeoM.Translate(-float64(w)/2.0, -float64(h)/2.0)
	op.GeoM.Scale(scaleX*zoom, scaleY*zoom)
	op.GeoM.Rotate(b.Rotation + math.Pi/2.0)
	cx := float64(b.X) + float64(b.Width)/2.0
	cy := float64(b.Y) + float64(b.Height)/2.0
	tx := (cx - camX) * zoom
	ty := (cy - camY) * zoom
	op.GeoM.Translate(math.Round(tx), math.Round(ty))
	op.Filter = ebiten.FilterNearest
	screen.DrawImage(img, op)
}

// Hitboxes implements component.DamageDealerComponent.
func (b *Bullet) Hitboxes() []component.Hitbox { return b.hitboxes }

// SetHitboxes implements component.DamageDealerComponent.
func (b *Bullet) SetHitboxes(boxes []component.Hitbox) { b.hitboxes = boxes }

// DamageFaction implements component.DamageDealerComponent.
func (b *Bullet) DamageFaction() component.Faction { return b.faction }

// EmitHit implements component.DamageDealerComponent.
func (b *Bullet) EmitHit(evt component.CombatEvent) { b.combatEmitter.Emit(evt) }

func (b *Bullet) updateHitbox() {
	if b == nil {
		return
	}
	hb := component.Hitbox{
		ID:      fmt.Sprintf("bullet_%d", b.ID),
		Rect:    common.Rect{X: b.X, Y: b.Y, Width: b.Width, Height: b.Height},
		Active:  true,
		OwnerID: b.OwnerID,
		Damage: component.Damage{
			Amount:         bulletDamageAmount,
			KnockbackX:     bulletKnockbackX,
			KnockbackY:     bulletKnockbackY,
			HitstunFrames:  6,
			CooldownFrames: bulletCooldownFrames,
			IFrameFrames:   bulletIFrameFrames,
			Faction:        b.faction,
			MultiHit:       false,
		},
	}
	b.hitboxes = []component.Hitbox{hb}
}

func (b *Bullet) overlapsPlayer(player *Player) bool {
	if b == nil || player == nil {
		return false
	}
	boxes := player.Hurtboxes()
	if len(boxes) == 0 {
		return false
	}
	br := common.Rect{X: b.X, Y: b.Y, Width: b.Width, Height: b.Height}
	for _, hu := range boxes {
		if !hu.Enabled {
			continue
		}
		if br.Intersects(&hu.Rect) {
			return true
		}
	}
	return false
}

func (b *Bullet) collidesWithPhysics(collisionWorld *CollisionWorld) bool {
	if b == nil || collisionWorld == nil || collisionWorld.level == nil {
		return false
	}
	level := collisionWorld.level
	left := int(math.Floor(float64(b.X) / float64(common.TileSize)))
	top := int(math.Floor(float64(b.Y) / float64(common.TileSize)))
	right := int(math.Floor(float64(b.X+b.Width-1) / float64(common.TileSize)))
	bottom := int(math.Floor(float64(b.Y+b.Height-1) / float64(common.TileSize)))
	if right < 0 || bottom < 0 || left >= level.Width || top >= level.Height {
		return true
	}
	if left < 0 {
		left = 0
	}
	if top < 0 {
		top = 0
	}
	if right >= level.Width {
		right = level.Width - 1
	}
	if bottom >= level.Height {
		bottom = level.Height - 1
	}
	for y := top; y <= bottom; y++ {
		for x := left; x <= right; x++ {
			if level.physicsTileAt(x, y) {
				return true
			}
		}
	}
	return false
}
