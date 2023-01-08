package system

import "math"

func AbsInt(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func AbsReal(r float64) float64 {
	return math.Abs(r)
}

func Arctan(r float64) float64 {
	return math.Atan(r)
}
