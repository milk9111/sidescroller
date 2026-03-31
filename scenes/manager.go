package scenes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Manager struct {
	factories  map[string]Factory
	active     Scene
	activeName string
}

func NewManager(initialScene string, factories map[string]Factory) (*Manager, error) {
	if len(factories) == 0 {
		return nil, fmt.Errorf("no scenes registered")
	}

	manager := &Manager{
		factories: make(map[string]Factory, len(factories)),
	}
	for name, factory := range factories {
		manager.factories[name] = factory
	}

	if initialScene == "" {
		initialScene = SceneGame
	}

	if err := manager.SwitchTo(initialScene); err != nil {
		return nil, err
	}

	return manager, nil
}

func (m *Manager) SwitchTo(name string) error {
	factory, ok := m.factories[name]
	if !ok {
		return fmt.Errorf("unknown scene %q", name)
	}
	if factory == nil {
		return fmt.Errorf("scene %q has no factory", name)
	}

	scene, err := factory()
	if err != nil {
		return fmt.Errorf("create scene %q: %w", name, err)
	}

	m.active = scene
	m.activeName = name
	return nil
}

func (m *Manager) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyF12) {
		return ErrQuit
	}

	if m.active == nil {
		return fmt.Errorf("no active scene")
	}

	nextScene, err := m.active.Update()
	if err != nil {
		return err
	}
	if nextScene != "" && nextScene != m.activeName {
		return m.SwitchTo(nextScene)
	}

	return nil
}

func (m *Manager) Draw(screen *ebiten.Image) {
	if m.active != nil {
		m.active.Draw(screen)
	}
}

func (m *Manager) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	if m.active != nil {
		return m.active.LayoutF(outsideWidth, outsideHeight)
	}
	return outsideWidth, outsideHeight
}

func (m *Manager) Layout(outsideWidth, outsideHeight int) (int, int) {
	if m.active != nil {
		return m.active.Layout(outsideWidth, outsideHeight)
	}
	return outsideWidth, outsideHeight
}

func (m *Manager) ActiveSceneName() string {
	return m.activeName
}
