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

func Cos(r float64) float64 {
	return math.Cos(r)
}

func Exp(r float64) float64 {
	return math.Exp(r)
}

func Frac(r float64) float64 {
	_, f := math.Modf(r)
	return f
}

func Int(r float64) float64 {
	f, _ := math.Modf(r)
	return f
}

func Ln(r float64) float64 {
	return math.Log(r)
}

func Pi() float64 {
	return math.Pi
}

func Sin(r float64) float64 {
	return math.Sin(r)
}

func Sqr(r float64) float64 {
	return r * r
}

func Sqrt(r float64) float64 {
	return math.Sqrt(r)
}

func Trunc(r float64) int {
	return int(r)
}

func Round(r float64) int {
	return Trunc(r + 0.5)
}

func Chr(i int) byte {
	return byte(i)
}

func Odd(i int) bool {
	return i%2 != 0
}
