package system

import (
	"testing"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

// BenchmarkStaticTileBatchSignature measures the cost of computing the
// signature used to detect changes in the static tile batch.
func BenchmarkStaticTileBatchSignature(b *testing.B) {
	const n = 2000
	w := ecs.NewWorld()

	for i := 0; i < n; i++ {
		e := ecs.CreateEntity(w)
		_ = ecs.Add(w, e, component.StaticTileComponent.Kind(), &component.StaticTile{})
		_ = ecs.Add(w, e, component.TransformComponent.Kind(), &component.Transform{X: float64(i), Y: float64(i % 10)})
		_ = ecs.Add(w, e, component.SpriteComponent.Kind(), &component.Sprite{Disabled: i%2 == 0})
		_ = ecs.Add(w, e, component.RenderLayerComponent.Kind(), &component.RenderLayer{Index: i % 8})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = staticTileBatchSignature(w)
	}
}
