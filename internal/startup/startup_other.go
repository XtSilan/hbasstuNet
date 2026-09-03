//go:build !windows

package startup

import "errors"

func Set(bool) error { return errors.New("automatic startup is only supported on Windows") }

func SyncCurrentPath() error { return nil }

func Enabled() (bool, error) { return false, nil }
