package common

func Lerp(a, b, t float32) float32 {
	return a + t*(b-a)
}
