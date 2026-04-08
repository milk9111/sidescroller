package component

type TransitionInput struct {
	UpPressed    bool
	UsingGamepad bool
}

var TransitionInputComponent = NewComponent[TransitionInput]()
