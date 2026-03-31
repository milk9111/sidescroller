package scenes

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/milk9111/sidescroller/common"
)

type TestScene struct {
	frames int
}

func NewTestScene() *TestScene {
	return &TestScene{}
}

func (s *TestScene) Update() (string, error) {
	s.frames++
	if s.frames >= 60 {
		return SceneGame, nil
	}
	return "", nil
}

func (s *TestScene) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "test")
}

func (s *TestScene) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	return common.BaseWidth, common.BaseHeight
}

func (s *TestScene) Layout(outsideWidth, outsideHeight int) (int, int) {
	return common.BaseWidth, common.BaseHeight
}
