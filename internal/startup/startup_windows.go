//go:build windows

package startup

import (
	"golang.org/x/sys/windows/registry"
	"os"
)

const keyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const valueName = "hbasstuNet"

func Set(enabled bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !enabled {
		return k.DeleteValue(valueName)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(valueName, `"`+exe+`" --background`)
}
