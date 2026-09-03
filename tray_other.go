//go:build !windows

package main

func (a *App) startTray()          {}
func (a *App) stopTray()           {}
func (a *App) updateTray(AppState) {}
