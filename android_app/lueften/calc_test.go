package main

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

const float64EqualityThreshold = 1e-6

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) <= float64EqualityThreshold
}

func TestCalcAbsHum(t *testing.T) {
	assert.True(t, almostEqual(7.05446191031343, calcAbsHum(80, 9)))
	assert.True(t, almostEqual(2.21233063513978, calcAbsHum(95, -9)))
}
