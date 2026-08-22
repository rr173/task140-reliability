package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors used across packages. HTTP layer maps these to status codes.
var (
	ErrNotFound          = errors.New("not found")
	ErrStateConflict     = errors.New("state conflict")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrDependentMissing  = errors.New("dependent resource missing")
	ErrInsufficientData  = errors.New("insufficient data")
	ErrRBDCannotReduce   = errors.New("rbd cannot reduce")
	ErrNotCoherent        = errors.New("fault tree not coherent")
	ErrAlreadyExists     = errors.New("already exists")
)

// ValidateRating asserts an FMEA rating is in [1,10].
func ValidateRating(name string, v int) error {
	if v < 1 || v > 10 {
		return fmt.Errorf("%w: %s must be 1..10, got %d", ErrInvalidArgument, name, v)
	}
	return nil
}

// ProbMicroToFloat converts a ProbMicro (1e-6) to a float64 probability.
func ProbMicroToFloat(p ProbMicro) float64 {
	return float64(p) / 1e6
}

// FloatToProbMicro converts a float64 probability to ProbMicro, clamped to [0,1e6].
func FloatToProbMicro(p float64) ProbMicro {
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	return ProbMicro(p * 1e6)
}

// FailureRateToProb converts a failure rate (1e-12/h integer) and a mission
// time (hours) to a probability q = 1 - exp(-lambda*t), returned as ProbMicro.
// For small lambda*t this is ~ lambda*t. Clamp to [0,1].
func FailureRateToProb(ratePPT FailureRatePPT, hours int64) ProbMicro {
	if ratePPT <= 0 || hours <= 0 {
		return 0
	}
	lambda := float64(ratePPT) * 1e-12
	q := 1.0
	lt := lambda * float64(hours)
	if lt < 1e-3 {
		q = lt // tiny: linear is accurate and avoids exp drift
	} else {
		q = 1 - expNeg(lt)
	}
	return FloatToProbMicro(q)
}

// expNeg returns exp(-x) without importing math at call sites that prefer a
// named helper. Implemented via math.Exp in math.go.
