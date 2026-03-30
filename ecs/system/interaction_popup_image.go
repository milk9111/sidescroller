package system

import "github.com/hajimehoshi/ebiten/v2"

func composePopupImage(base, cue *ebiten.Image) *ebiten.Image {
	if base == nil {
		return nil
	}

	image := ebiten.NewImageFromImage(base)
	if cue != nil {
		image.DrawImage(cue, &ebiten.DrawImageOptions{})
	}

	return image
}
