// Package util provides common utility functions for the CaptchaX application.
// It includes mathematical operations, random number generation, and other
// commonly used helper functions.
package util

import (
	"crypto/rand"
	"math/big"
)

// AbsInt returns the absolute value of an integer.
func AbsInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// MaxInt returns the larger of two integers.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MinInt returns the smaller of two integers.
func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RandomInt returns a random integer in the range [min, max] (inclusive).
// Uses crypto/rand for secure random number generation.
func RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return min
	}
	return int(n.Int64()) + min
}

// ClampInt constrains a value to be within the range [min, max].
// If val is less than min, returns min.
// If val is greater than max, returns max.
// Otherwise, returns val unchanged.
func ClampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
