package component

// AnimationEventType identifies a type of frame event.
type AnimationEventType string

const (
	AnimationEventEnableHitbox  AnimationEventType = "enable_hitbox"
	AnimationEventDisableHitbox AnimationEventType = "disable_hitbox"
	AnimationEventEmit          AnimationEventType = "emit"
)

// AnimationEvent is emitted by animation frame callbacks.
type AnimationEvent struct {
	Type     AnimationEventType
	HitboxID string
	Payload  string
}

// AnimationEventHandler handles animation frame events.
type AnimationEventHandler func(anim *Animation, frame int, evt AnimationEvent)

// AnimationEventEmitter dispatches animation frame events to handlers.
type AnimationEventEmitter struct {
	Handlers []AnimationEventHandler
}

// Emit sends a frame event to all handlers.
func (e *AnimationEventEmitter) Emit(anim *Animation, frame int, evt AnimationEvent) {
	if e == nil || len(e.Handlers) == 0 {
		return
	}
	for _, h := range e.Handlers {
		if h != nil {
			h(anim, frame, evt)
		}
	}
}

// AnimationEventMap stores per-frame events.
type AnimationEventMap struct {
	Frames map[int][]AnimationEvent
}

// NewAnimationEventMap creates a new event map.
func NewAnimationEventMap() *AnimationEventMap {
	return &AnimationEventMap{Frames: make(map[int][]AnimationEvent)}
}

// Add adds an event for a frame.
func (m *AnimationEventMap) Add(frame int, evt AnimationEvent) {
	if m == nil || frame < 0 {
		return
	}
	if m.Frames == nil {
		m.Frames = make(map[int][]AnimationEvent)
	}
	m.Frames[frame] = append(m.Frames[frame], evt)
}

// BindAnimationEvents registers callbacks on the animation to emit events for frames.
func BindAnimationEvents(anim *Animation, events *AnimationEventMap, emitter *AnimationEventEmitter) {
	if anim == nil || events == nil || len(events.Frames) == 0 {
		return
	}
	for frame, evts := range events.Frames {
		f := frame
		copied := append([]AnimationEvent(nil), evts...)
		anim.AddFrameCallback(f, func(a *Animation, frameIdx int) {
			for _, evt := range copied {
				if emitter != nil {
					emitter.Emit(a, frameIdx, evt)
				}
			}
		})
	}
}
