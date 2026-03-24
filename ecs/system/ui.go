package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type UISystem struct{}

func NewUISystem() *UISystem {
	return &UISystem{}
}

func (s *UISystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	ent, ok := ecs.First(w, component.UIRootComponent.Kind())
	if !ok {
		return
	}

	root, ok := ecs.Get(w, ent, component.UIRootComponent.Kind())
	if !ok || root == nil || root.UI == nil {
		return
	}

	root.UI.Update()
}

func (s *UISystem) Draw(w *ecs.World, screen *ebiten.Image) {
	if w == nil || screen == nil {
		return
	}

	ent, ok := ecs.First(w, component.UIRootComponent.Kind())
	if !ok {
		return
	}

	root, ok := ecs.Get(w, ent, component.UIRootComponent.Kind())
	if !ok || root == nil || root.UI == nil {
		return
	}

	root.UI.Draw(screen)
}
