package component

type DialogueInput struct {
	Pressed      bool
	UsingGamepad bool
}

var DialogueInputComponent = NewComponent[DialogueInput]()
