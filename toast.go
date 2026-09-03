package main

import (
	"log"
	"os"
	"path/filepath"

	toast "git.sr.ht/~jackmordaunt/go-toast/v2"
)

func initToast() {
	executable, err := os.Executable()
	if err != nil {
		log.Printf("toast setup failed: %v", err)
		return
	}
	if err := toast.SetAppData(toast.AppData{
		AppID:    "hbasstuNet",
		IconPath: filepath.Join(filepath.Dir(executable), "tray.ico"),
	}); err != nil {
		log.Printf("toast setup failed: %v", err)
	}
}

func showToast(title, body string) {
	notification := toast.Notification{AppID: "hbasstuNet", Title: title, Body: body}
	if err := notification.Push(); err != nil {
		log.Printf("toast notification failed: %v", err)
	}
}
