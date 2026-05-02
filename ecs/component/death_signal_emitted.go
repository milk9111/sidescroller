package component

// DeathSignalEmitted marks entities that have already had their on_death
// signal emitted for the current zero-health state.
type DeathSignalEmitted struct{}

var DeathSignalEmittedComponent = NewComponent[DeathSignalEmitted]()
