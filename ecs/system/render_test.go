package system

import (
	"image"
	"sort"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

func TestDrawLayerIndexPrefersEntityLayer(t *testing.T) {
	w := ecs.NewWorld()
	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 7})
	_ = ecs.Add(w, e, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: 2})

	if got := drawLayerIndex(w, e); got != 2 {
		t.Fatalf("expected draw layer 2, got %d", got)
	}
	if got := renderOrderIndex(w, e); got != 7 {
		t.Fatalf("expected render order 7, got %d", got)
	}
}

func TestEntityLayerScopesRenderOrder(t *testing.T) {
	w := ecs.NewWorld()
	backHighOrder := ecs.CreateEntity(w)
	frontLowOrder := ecs.CreateEntity(w)
	for _, tc := range []struct {
		entity     ecs.Entity
		layerIndex int
		orderIndex int
	}{
		{entity: backHighOrder, layerIndex: 0, orderIndex: 99},
		{entity: frontLowOrder, layerIndex: 1, orderIndex: 0},
	} {
		_ = ecs.Add(w, tc.entity, component.EntityLayerComponent.Kind(), &component.EntityLayer{Index: tc.layerIndex})
		_ = ecs.Add(w, tc.entity, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: tc.orderIndex})
	}

	entities := []ecs.Entity{frontLowOrder, backHighOrder}
	sort.Slice(entities, func(i, j int) bool {
		li := drawLayerIndex(w, entities[i])
		lj := drawLayerIndex(w, entities[j])
		if li != lj {
			return li < lj
		}
		oi := renderOrderIndex(w, entities[i])
		oj := renderOrderIndex(w, entities[j])
		if oi != oj {
			return oi < oj
		}
		return uint64(entities[i]) < uint64(entities[j])
	})

	if entities[0] != backHighOrder {
		t.Fatalf("expected lower entity layer to sort first regardless of render order")
	}
}

func TestClampViewToLevelBounds(t *testing.T) {
	bounds := &component.LevelBounds{Width: 320, Height: 180}
	left, top, right, bottom := clampViewToLevelBounds(bounds, -40, -10, 360, 200)
	if left != 0 || top != 0 || right != 320 || bottom != 180 {
		t.Fatalf("expected clamped view to match level bounds, got (%v,%v,%v,%v)", left, top, right, bottom)
	}
}

func TestWorldClipRectClipsToProjectedLevelBounds(t *testing.T) {
	screenBounds := image.Rect(0, 0, 640, 360)
	bounds := &component.LevelBounds{Width: 200, Height: 100}
	clip, ok := worldClipRect(screenBounds, bounds, 50, 25, 2)
	if !ok {
		t.Fatal("expected clip rect to exist")
	}
	want := image.Rect(0, 0, 300, 150)
	if clip != want {
		t.Fatalf("expected clip rect %v, got %v", want, clip)
	}
}

func TestWorldClipRectAllowsFullScreenWithoutBounds(t *testing.T) {
	screenBounds := image.Rect(0, 0, 640, 360)
	clip, ok := worldClipRect(screenBounds, nil, 0, 0, 1)
	if !ok {
		t.Fatal("expected full-screen clip rect")
	}
	if clip != screenBounds {
		t.Fatalf("expected full screen clip rect %v, got %v", screenBounds, clip)
	}
}

func TestEnsureStaticTileBatchRebuildsWhenStaticTilesChange(t *testing.T) {
	w := ecs.NewWorld()
	r := NewRenderSystem()
	img := ebiten.NewImage(32, 32)

	tile := ecs.CreateEntity(w)
	_ = ecs.Add(w, tile, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, tile, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, tile, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
	_ = ecs.Add(w, tile, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 0})

	r.ensureStaticTileBatch(w)
	if got := len(r.batch.chunks); got != 1 {
		t.Fatalf("expected one static chunk after initial build, got %d", got)
	}

	second := ecs.CreateEntity(w)
	_ = ecs.Add(w, second, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, second, component.TransformComponent.Kind(), &component.Transform{X: 600, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, second, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
	_ = ecs.Add(w, second, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 0})

	r.ensureStaticTileBatch(w)
	if got := len(r.batch.chunks); got != 2 {
		t.Fatalf("expected two static chunks after adding a tile in a new chunk, got %d", got)
	}
}

func TestBuildStaticTileBatchSkipsDisabledTiles(t *testing.T) {
	w := ecs.NewWorld()
	r := NewRenderSystem()
	r.batch = staticTileBatch{world: w, chunkSize: 512}
	img := ebiten.NewImage(32, 32)

	tile := ecs.CreateEntity(w)
	_ = ecs.Add(w, tile, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, tile, component.TransformComponent.Kind(), &component.Transform{X: 0, Y: 0, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, tile, component.SpriteComponent.Kind(), &component.Sprite{Image: img, Disabled: true})
	_ = ecs.Add(w, tile, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 0})

	r.buildStaticTileBatch(w)
	if got := len(r.batch.chunks); got != 0 {
		t.Fatalf("expected disabled static tiles to be excluded from the batch, got %d chunks", got)
	}
}

func TestDrawAppliesSpriteShakeToStaticTiles(t *testing.T) {
	w := ecs.NewWorld()
	img := ebiten.NewImage(1, 1)

	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.StaticTileComponent.Kind(), &component.StaticTile{})
	_ = ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{X: 2, Y: 3, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
	_ = ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 0})
	_ = ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{Frames: 4, Intensity: 2, OffsetX: 1, OffsetY: -1})

	tf, _ := ecs.Get(w, e, component.TransformComponent.Kind())
	sprite, _ := ecs.Get(w, e, component.SpriteComponent.Kind())
	geoM := spriteGeoM(w, e, tf, sprite, img)
	x, y := geoM.Apply(0, 0)
	if x != 3 || y != 2 {
		t.Fatalf("expected shaken sprite origin at (3,2), got (%v,%v)", x, y)
	}
}

func TestDrawAppliesSpriteShakeToPhysicsSprites(t *testing.T) {
	w := ecs.NewWorld()
	img := ebiten.NewImage(1, 1)

	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{X: 2, Y: 3, ScaleX: 1, ScaleY: 1})
	_ = ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{Image: img})
	_ = ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: 0})
	_ = ecs.Add(w, e, component.PhysicsBodyComponent.Kind(), &component.PhysicsBody{Width: 1, Height: 1, OffsetX: 0, OffsetY: 0})
	_ = ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{Frames: 4, Intensity: 2, OffsetX: 1, OffsetY: -1})

	tf, _ := ecs.Get(w, e, component.TransformComponent.Kind())
	sprite, _ := ecs.Get(w, e, component.SpriteComponent.Kind())
	geoM := spriteGeoM(w, e, tf, sprite, img)
	x, y := geoM.Apply(0, 0)
	if x != 3 || y != 2 {
		t.Fatalf("expected shaken physics sprite origin at (3,2), got (%v,%v)", x, y)
	}
}

func TestDrawAreaTileAppliesSpriteShake(t *testing.T) {
	w := ecs.NewWorld()

	e := ecs.CreateEntity(w)
	_ = ecs.Add(w, e, component.SpriteShakeComponent.Kind(), &component.SpriteShake{Frames: 4, Intensity: 2, OffsetX: 1, OffsetY: -1})

	centerX, centerY := areaTileCenter(w, e, 2, 3, 1, 1)
	if centerX != 3.5 || centerY != 2.5 {
		t.Fatalf("expected shaken area tile center at (3.5,2.5), got (%v,%v)", centerX, centerY)
	}
}
