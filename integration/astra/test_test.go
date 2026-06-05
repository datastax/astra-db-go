package astra_test

import (
	"testing"
)

func TestName(t *testing.T) {
	t.Run("TestName", func(t *testing.T) {
		// This test is just a placeholder to verify that the test framework is set up correctly.
	})

	RunTestParallel(t, "TestNameParallel", func(t *testing.T) {
		// This test runs in parallel to verify that the RunTestParallel helper works correctly.
	})
}

func RunTestParallel(t *testing.T, funcName string, testFunc func(t *testing.T)) {
	t.Run(funcName, func(t *testing.T) {
		t.Parallel()
		testFunc(t)
	})
}
