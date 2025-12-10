package docker

import "fmt"

// FormatBytes formats bytes into human-readable format
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatBytesInt64 formats int64 bytes into human-readable format
func FormatBytesInt64(bytes int64) string {
	if bytes < 0 {
		return "0B"
	}
	return FormatBytes(uint64(bytes))
}

// FormatPercent formats a percentage value
func FormatPercent(percent float64) string {
	if percent < 0.01 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", percent)
}

// FormatNetIO formats network I/O statistics
func FormatNetIO(rx, tx uint64) string {
	return fmt.Sprintf("%s / %s", FormatBytes(rx), FormatBytes(tx))
}

// FormatBlockIO formats block I/O statistics
func FormatBlockIO(read, write uint64) string {
	return fmt.Sprintf("%s / %s", FormatBytes(read), FormatBytes(write))
}

// FormatMemUsage formats memory usage statistics
func FormatMemUsage(usage, limit uint64) string {
	return fmt.Sprintf("%s / %s", FormatBytes(usage), FormatBytes(limit))
}
