//go:build !windows

package main

import "fmt"

func launchUpdater(updatePath, executable string, pid int) error {
	return fmt.Errorf("自动更新仅支持 Windows")
}
