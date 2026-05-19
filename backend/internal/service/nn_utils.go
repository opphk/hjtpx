package service

import (
	"math"
	"math/rand"
)

func randomMatrix(size int) []float64 {
	mat := make([]float64, size)
	scale := math.Sqrt(2.0 / float64(size))
	for i := range mat {
		mat[i] = (rand.Float64() - 0.5) * scale
	}
	return mat
}

func randomVector(size int) []float64 {
	vec := make([]float64, size)
	for i := range vec {
		vec[i] = rand.Float64()*0.2 - 0.1
	}
	return vec
}

func softmax(x []float64, start, length int) []float64 {
	maxVal := x[start]
	for i := start + 1; i < start+length; i++ {
		if x[i] > maxVal {
			maxVal = x[i]
		}
	}
	sum := 0.0
	for i := start; i < start+length; i++ {
		x[i] = math.Exp(x[i] - maxVal)
		sum += x[i]
	}
	for i := start; i < start+length; i++ {
		x[i] /= sum
	}
	return x
}
