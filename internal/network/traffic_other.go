//go:build !windows

package network

import "fmt"

func Traffic(interfaceName string) (uint64, uint64, error) {
	return 0, 0, fmt.Errorf("traffic counters are only available on Windows")
}
