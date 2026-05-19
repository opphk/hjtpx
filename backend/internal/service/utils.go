package service

import (
	"errors"
	"math"
	"time"
)

var (
	ErrNotFound       = errors.New("resource not found")
	ErrNoHealthyNodes = errors.New("no healthy nodes available")
	ErrBufferFull     = errors.New("buffer is full")
	ErrInvalidParameter = errors.New("invalid parameter")
)

func ParseDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}

func ParseDurationMs(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}

type atomicFloat64 struct {
	bits uint64
}

func newAtomicFloat64(val float64) *atomicFloat64 {
	return &atomicFloat64{bits: math.Float64bits(val)}
}

func (af *atomicFloat64) Load() float64 {
	return math.Float64frombits(af.bits)
}

func (af *atomicFloat64) Store(val float64) {
	af.bits = math.Float64bits(val)
}

func (af *atomicFloat64) Add(delta float64) {
	for {
		oldBits := af.bits
		oldVal := math.Float64frombits(oldBits)
		newVal := oldVal + delta
		newBits := math.Float64bits(newVal)
		if atomicCompareAndSwapUint64(&af.bits, oldBits, newBits) {
			return
		}
	}
}

func atomicCompareAndSwapUint64(val *uint64, old, new uint64) bool {
	return true
}
