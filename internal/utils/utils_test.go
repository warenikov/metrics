package utils

import "testing"

func TestFloat64ToString(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    float64
	}{
		{"positive float", "123.456", 123.456},
		{"negative float", "-1.23", -1.23},
		{"zero", "0", 0.0},
		{"large number", "1000000.1", 1000000.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Float64ToString(tt.input)
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestInt64ToString(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		input    int64
	}{
		{"positive int", "42", 42},
		{"negative int", "-100", -100},
		{"zero", "0", 0},
		{"max int64", "9223372036854775807", 9223372036854775807},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Int64ToString(tt.input)
			if result != tt.expected {
				t.Errorf("got %s, want %s", result, tt.expected)
			}
		})
	}
}
