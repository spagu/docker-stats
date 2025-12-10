package docker

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    uint64
		expected string
	}{
		{"zero bytes", 0, "0B"},
		{"bytes", 500, "500B"},
		{"kilobytes", 1024, "1.0KiB"},
		{"megabytes", 1048576, "1.0MiB"},
		{"gigabytes", 1073741824, "1.0GiB"},
		{"terabytes", 1099511627776, "1.0TiB"},
		{"mixed kilobytes", 1536, "1.5KiB"},
		{"mixed megabytes", 1572864, "1.5MiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %s; want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesInt64(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"negative", -100, "0B"},
		{"zero", 0, "0B"},
		{"positive", 1024, "1.0KiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytesInt64(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytesInt64(%d) = %s; want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected string
	}{
		{"zero", 0, "0.00%"},
		{"small", 0.001, "0.00%"},
		{"normal", 50.5, "50.50%"},
		{"high", 99.99, "99.99%"},
		{"over 100", 150.0, "150.00%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPercent(tt.percent)
			if result != tt.expected {
				t.Errorf("FormatPercent(%f) = %s; want %s", tt.percent, result, tt.expected)
			}
		})
	}
}

func TestFormatNetIO(t *testing.T) {
	tests := []struct {
		name     string
		rx       uint64
		tx       uint64
		expected string
	}{
		{"zero", 0, 0, "0B / 0B"},
		{"bytes", 100, 200, "100B / 200B"},
		{"mixed", 1024, 2048, "1.0KiB / 2.0KiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatNetIO(tt.rx, tt.tx)
			if result != tt.expected {
				t.Errorf("FormatNetIO(%d, %d) = %s; want %s", tt.rx, tt.tx, result, tt.expected)
			}
		})
	}
}

func TestFormatBlockIO(t *testing.T) {
	tests := []struct {
		name     string
		read     uint64
		write    uint64
		expected string
	}{
		{"zero", 0, 0, "0B / 0B"},
		{"bytes", 100, 200, "100B / 200B"},
		{"mixed", 1048576, 2097152, "1.0MiB / 2.0MiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBlockIO(tt.read, tt.write)
			if result != tt.expected {
				t.Errorf("FormatBlockIO(%d, %d) = %s; want %s", tt.read, tt.write, result, tt.expected)
			}
		})
	}
}

func TestFormatMemUsage(t *testing.T) {
	tests := []struct {
		name     string
		usage    uint64
		limit    uint64
		expected string
	}{
		{"zero", 0, 0, "0B / 0B"},
		{"normal", 1073741824, 2147483648, "1.0GiB / 2.0GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMemUsage(tt.usage, tt.limit)
			if result != tt.expected {
				t.Errorf("FormatMemUsage(%d, %d) = %s; want %s", tt.usage, tt.limit, result, tt.expected)
			}
		})
	}
}

func TestTrimContainerName(t *testing.T) {
	tests := []struct {
		name     string
		names    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"with slash", []string{"/mycontainer"}, "mycontainer"},
		{"without slash", []string{"mycontainer"}, "mycontainer"},
		{"multiple names", []string{"/first", "/second"}, "first"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimContainerName(tt.names)
			if result != tt.expected {
				t.Errorf("trimContainerName(%v) = %s; want %s", tt.names, result, tt.expected)
			}
		})
	}
}

func TestSortContainers(t *testing.T) {
	containers := []ContainerStats{
		{Name: "alpha", CPUPercent: 50, MemPercent: 30},
		{Name: "beta", CPUPercent: 10, MemPercent: 80},
		{Name: "gamma", CPUPercent: 90, MemPercent: 10},
	}

	t.Run("sort by name ascending", func(t *testing.T) {
		c := make([]ContainerStats, len(containers))
		copy(c, containers)
		SortContainers(c, SortByName, true)
		if c[0].Name != "alpha" || c[1].Name != "beta" || c[2].Name != "gamma" {
			t.Errorf("Sort by name ascending failed: %v", c)
		}
	})

	t.Run("sort by CPU descending", func(t *testing.T) {
		c := make([]ContainerStats, len(containers))
		copy(c, containers)
		SortContainers(c, SortByCPU, false)
		if c[0].Name != "gamma" || c[1].Name != "alpha" || c[2].Name != "beta" {
			t.Errorf("Sort by CPU descending failed: %v", c)
		}
	})

	t.Run("sort by memory descending", func(t *testing.T) {
		c := make([]ContainerStats, len(containers))
		copy(c, containers)
		SortContainers(c, SortByMemory, false)
		if c[0].Name != "beta" || c[1].Name != "alpha" || c[2].Name != "gamma" {
			t.Errorf("Sort by memory descending failed: %v", c)
		}
	})
}
