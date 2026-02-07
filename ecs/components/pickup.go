package components

// Pickup stores data for ability pickups.
type Pickup struct {
	Kind      string
	Enabled   bool
	BaseX     float32
	BaseY     float32
	Phase     float64
	Amplitude float32
	Frequency float32
	Width     float32
	Height    float32
}
