package common

import "math"

func Lerp(a, b, t float32) float32 {
	return a + t*(b-a)
}

// Distance returns the Euclidean distance between (x1,y1) and (x2,y2).
func Distance(x1, y1, x2, y2 float64) float64 {
	return math.Hypot(x2-x1, y2-y1)
}
