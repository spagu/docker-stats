package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestGetCPUColor(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected tcell.Color
	}{
		{"low usage", 10, tcell.ColorWhite},
		{"medium-low usage", 25, tcell.ColorGreen},
		{"medium usage", 55, tcell.ColorYellow},
		{"high usage", 85, tcell.ColorRed},
		{"very high usage", 100, tcell.ColorRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCPUColor(tt.percent)
			if result != tt.expected {
				t.Errorf("getCPUColor(%f) = %v; want %v", tt.percent, result, tt.expected)
			}
		})
	}
}

func TestGetMemColor(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected tcell.Color
	}{
		{"low usage", 30, tcell.ColorWhite},
		{"medium-low usage", 45, tcell.ColorGreen},
		{"medium usage", 75, tcell.ColorYellow},
		{"high usage", 95, tcell.ColorRed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMemColor(tt.percent)
			if result != tt.expected {
				t.Errorf("getMemColor(%f) = %v; want %v", tt.percent, result, tt.expected)
			}
		})
	}
}
