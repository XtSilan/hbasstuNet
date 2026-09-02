//go:build windows

package startup

import (
	"errors"
	"golang.org/x/sys/windows/registry"
	"os"
	"syscall"
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
		err := k.DeleteValue(valueName)
		if ignoreMissingValue(err) == nil {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return k.SetStringValue(valueName, `"`+exe+`" --background`)
}

func ignoreMissingValue(err error) error {
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return err
}
