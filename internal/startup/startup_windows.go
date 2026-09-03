//go:build windows

package startup

import (
	"errors"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const keyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
const valueName = "hbasstuNet"
const approvedPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`

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

// Enabled reports whether Windows currently considers the startup entry
// enabled, including the StartupApproved state controlled by Task Manager.
func Enabled() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return false, nil
		}
		return false, err
	}
	_, _, err = k.GetStringValue(valueName)
	k.Close()
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return false, nil
		}
		return false, err
	}
	approved, err := registry.OpenKey(registry.CURRENT_USER, approvedPath, registry.QUERY_VALUE)
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return true, nil
		}
		return false, err
	}
	defer approved.Close()
	data, _, err := approved.GetBinaryValue(valueName)
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return true, nil
		}
		return false, err
	}
	return len(data) == 0 || data[0] == 0x02, nil
}

// SyncCurrentPath repairs a moved portable executable without creating a new
// startup entry when the user disabled or removed it in Task Manager.
func SyncCurrentPath() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return nil
		}
		return err
	}
	defer k.Close()
	value, _, err := k.GetStringValue(valueName)
	if err != nil {
		if ignoreMissingValue(err) == nil {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	desired := `"` + exe + `" --background`
	if strings.TrimSpace(value) == desired {
		return nil
	}
	return k.SetStringValue(valueName, desired)
}

func ignoreMissingValue(err error) error {
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return err
}
