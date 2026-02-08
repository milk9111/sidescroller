package system

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

type InputSystem struct{}

func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

func (i *InputSystem) Update(w *ecs.World) {
	if w == nil {
		return
	}

	left := ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft)
	right := ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight)
	jump := ebiten.IsKeyPressed(ebiten.KeySpace)
	jumpPressed := inpututil.IsKeyJustPressed(ebiten.KeySpace)

	moveX := 0.0
	if left {
		moveX -= 1
	}
	if right {
		moveX += 1
	}

	for _, e := range w.Query(component.InputComponent.Kind()) {
		input, ok := ecs.Get(w, e, component.InputComponent)
		if !ok {
			input = component.Input{}
		}
		input.MoveX = moveX
		input.Jump = jump
		input.JumpPressed = jumpPressed
		if err := ecs.Add(w, e, component.InputComponent, input); err != nil {
			panic("input system: update input: " + err.Error())
		}
	}
}
