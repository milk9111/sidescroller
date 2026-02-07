package render

import "github.com/milk9111/sidescroller/component"

// AnimationClip stores an animation and optional event map.
type AnimationClip struct {
	Anim   *component.Animation
	Events *component.AnimationEventMap
}

// AnimationLibrary stores animation clips by key.
type AnimationLibrary struct {
	clips map[string]AnimationClip
}

// NewAnimationLibrary creates an empty library.
func NewAnimationLibrary() *AnimationLibrary {
	return &AnimationLibrary{clips: make(map[string]AnimationClip)}
}

// Register adds an animation clip to the library.
func (l *AnimationLibrary) Register(key string, anim *component.Animation, events *component.AnimationEventMap) {
	if l == nil || key == "" || anim == nil {
		return
	}
	l.clips[key] = AnimationClip{Anim: anim, Events: events}
}

// Get returns an animation clip by key.
func (l *AnimationLibrary) Get(key string) (AnimationClip, bool) {
	if l == nil || key == "" {
		return AnimationClip{}, false
	}
	clip, ok := l.clips[key]
	return clip, ok
}
