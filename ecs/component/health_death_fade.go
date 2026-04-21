package component

// HealthDeathFade tracks entities that have reached zero health and are
// waiting to fade out before being destroyed.
type HealthDeathFade struct {
	FadeFrames          int
	PostAnimationFrames int
	PostAnimationArmed  bool
	FadeStarted         bool
}

var HealthDeathFadeComponent = NewComponent[HealthDeathFade]()
