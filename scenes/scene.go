package scenes

import "github.com/hajimehoshi/ebiten/v2"

const (
	SceneStartMenu = "start_menu"
	SceneGame      = "game"
	SceneTest      = "test"
	SceneIntro     = "intro"
)

type Scene interface {
	Update() (string, error)
	Draw(screen *ebiten.Image)
	LayoutF(outsideWidth, outsideHeight float64) (float64, float64)
	Layout(outsideWidth, outsideHeight int) (int, int)
}

type Factory func() (Scene, error)
