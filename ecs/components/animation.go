package components

import "github.com/milk9111/sidescroller/component"

// Animator stores a runtime animation and optional events.
type Animator struct {
	ClipKey  string
	Anim     *component.Animation
	EventMap *component.AnimationEventMap
	Emitter  *component.AnimationEventEmitter
	Playing  bool
}
