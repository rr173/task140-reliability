package domain

import "math"

// expNeg returns exp(-x).
func expNeg(x float64) float64 { return math.Exp(-x) }

// Gamma uses the standard library gamma function (Go 1.26 has math.Gamma).
func gamma(x float64) float64 { return math.Gamma(x) }

// Gamma exposes math.Gamma to other packages in this module (Weibull MTTF).
func Gamma(x float64) float64 { return gamma(x) }

// Ln returns the natural log.
func Ln(x float64) float64 { return math.Log(x) }

// Log1pNeg returns log(1+x) with x possibly making 1+x approach 0 from below
// for Weibull plotting positions; safe-guarded.
func LogNeg(x float64) float64 { return math.Log(x) }

// Pow is math.Pow.
func Pow(base, exp float64) float64 { return math.Pow(base, exp) }

// Exp is math.Exp.
func Exp(x float64) float64 { return math.Exp(x) }

// Sqrt is math.Sqrt.
func Sqrt(x float64) float64 { return math.Sqrt(x) }

// RoundHalfUp rounds half away from zero to an integer. Used for integer
// presentation of availability/percent fields.
func RoundHalfUp(f float64) int64 {
	if f >= 0 {
		return int64(f + 0.5)
	}
	return int64(f - 0.5)
}

// RoundN rounds f to n decimal places (half away from zero) and returns a
// float64 truncated to that precision. Used for JSON presentation.
func RoundN(f float64, n int) float64 {
	p := math.Pow(10, float64(n))
	r := f * p
	if r >= 0 {
		r = math.Floor(r + 0.5)
	} else {
		r = math.Ceil(r - 0.5)
	}
	return r / p
}
