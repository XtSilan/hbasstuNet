//go:build windows

package main

import (
	_ "embed"
	"log"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed tray.ico
var trayIcon []byte

var trayQuit func()
var trayAccount *systray.MenuItem

func (a *App) startTray() {
	systray.Run(func() {
		systray.SetIcon(trayIcon)
		systray.SetTooltip("hbasstuNet 校园网登录器")
		trayAccount = systray.AddMenuItem("未连接校园网", "当前连接状态")
		systray.AddSeparator()
		show := systray.AddMenuItem("显示主界面", "打开 hbasstuNet")
		disconnect := systray.AddMenuItem("断开网络", "断开校园网认证")
		settings := systray.AddMenuItem("设置", "打开应用设置")
		systray.AddSeparator()
		quit := systray.AddMenuItem("退出", "退出 hbasstuNet")
		trayQuit = systray.Quit
		go func() {
			for {
				select {
				case <-show.ClickedCh:
					if a.ctx != nil {
						runtime.WindowShow(a.ctx)
					}
				case <-disconnect.ClickedCh:
					if err := a.Logout(); err != nil {
						log.Printf("tray logout failed: %v", err)
					}
				case <-settings.ClickedCh:
					if a.ctx != nil {
						runtime.WindowShow(a.ctx)
						runtime.EventsEmit(a.ctx, "navigate:settings")
					}
				case <-quit.ClickedCh:
					_ = a.ExitApp(false)
					return
				}
			}
		}()
		updateTray(a.State())
	}, func() {})
}

func (a *App) stopTray() {
	if trayQuit != nil {
		trayQuit()
	}
}

func (a *App) updateTray(state AppState) {
	updateTray(state)
}

func updateTray(state AppState) {
	if trayAccount == nil {
		return
	}
	if state.Status == "connected" {
		name := state.Account
		if name == "" {
			name = "已连接"
		}
		trayAccount.SetTitle("已连接：" + name)
		return
	}
	trayAccount.SetTitle("未连接校园网")
}
