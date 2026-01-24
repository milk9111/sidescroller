package main

type Rect struct {
	X, Y          float32
	Width, Height float32
}

func (r *Rect) Intersects(other *Rect) bool {
	return r.X < other.X+other.Width &&
		r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height &&
		r.Y+r.Height > other.Y
}
