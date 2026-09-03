package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	ISP       string `json:"isp"`
	Remember  bool   `json:"remember"`
	AutoLogin bool   `json:"autoLogin"`
	// AutoStart controls the Windows startup entry; AutoLogin controls login
	// after the app discovers a campus network.
	AutoStart      bool   `json:"autoStart,omitempty"`
	ExitBehavior   string `json:"exitBehavior"`
	SkipExitPrompt bool   `json:"skipExitPrompt"`
}

func Path() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "hbasstuNet", "settings.json")
	}
	return "settings.json"
}

func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Settings{Role: "student", ISP: "cucc"}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	if settings.Role == "" {
		settings.Role = "student"
	}
	if settings.ISP == "" {
		settings.ISP = "cucc"
	}
	if !settings.AutoLogin && settings.AutoStart {
		settings.AutoLogin = true
	}
	if settings.ExitBehavior != "exit" {
		settings.ExitBehavior = "tray"
	}
	if !settings.Remember {
		settings.Password = ""
	} else if settings.Password != "" {
		settings.Password, err = unprotect(settings.Password)
		if err != nil {
			return Settings{}, err
		}
	}
	return settings, nil
}

func Save(path string, settings Settings) error {
	if !settings.Remember {
		settings.Password = ""
	} else if settings.Password != "" {
		protected, err := protect(settings.Password)
		if err != nil {
			return err
		}
		settings.Password = protected
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
