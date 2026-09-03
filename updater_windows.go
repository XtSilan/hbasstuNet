//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

func launchUpdater(updatePath, executable string, pid int) error {
	command := fmt.Sprintf("$p=Get-Process -Id %s -ErrorAction SilentlyContinue; if($p){Wait-Process -Id %s}; Start-Sleep -Milliseconds 300; $u=%s; $e=%s; try {[System.IO.File]::Replace($u,$e,$null,$true)} catch {Move-Item -Force -LiteralPath $u -Destination $e}; Start-Process -FilePath $e", strconv.Itoa(pid), strconv.Itoa(pid), escapePS(updatePath), escapePS(executable))
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

func escapePS(path string) string { return "'" + replaceSingleQuotes(path) + "'" }

func replaceSingleQuotes(path string) string {
	result := ""
	for _, r := range path {
		if r == '\'' {
			result += "''"
		} else {
			result += string(r)
		}
	}
	return result
}
