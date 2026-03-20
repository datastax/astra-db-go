package main

import "testing"

func TestLowerFirst(t *testing.T) {
	// Created some tests juuuust in case we have some non-ASCII builder names
	// in the future. It is extremely unlikely, but, these are valid go identifiers.
	tests := []struct {
		name string
		want string
	}{
		{"CreateIndexOptionsBuilder", "createIndexOptionsBuilder"},
		{"ΔeltaBuilder", "δeltaBuilder"},
		{"运行Builder", "运行Builder"},
		{"πBuilder", "πBuilder"}, // pi builder!
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lowerFirst(tt.name); got != tt.want {
				t.Errorf("lowerFirst(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
