package ui

import (
	"fmt"
)

func formatUptime(seconds int64) string {
	days := seconds / 86400
	remaining := seconds % 86400
	hours := remaining / 3600
	remaining = remaining % 3600
	minutes := remaining / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
