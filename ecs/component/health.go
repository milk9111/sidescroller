package component

type Health struct {
	Initial int
	Current int
}

var HealthComponent = NewComponent[Health]()
