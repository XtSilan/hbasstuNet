//go:build !windows

package config

import "fmt"

func protect(string) (string, error) {
	return "", fmt.Errorf("password storage is only supported on Windows")
}

func unprotect(string) (string, error) {
	return "", fmt.Errorf("password storage is only supported on Windows")
}
