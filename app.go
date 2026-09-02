package main

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"hbasstuNet/internal/config"
	"hbasstuNet/internal/network"
	"hbasstuNet/internal/portal"
	"hbasstuNet/internal/startup"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	mu         sync.Mutex
	settings   config.Settings
	configPath string
	state      AppState
	client     *portal.Client
	info       network.Info
}

type AppState struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	SSID        string `json:"ssid"`
	Interface   string `json:"interface"`
	IP          string `json:"ip"`
	MAC         string `json:"mac"`
	Signal      string `json:"signal"`
	Account     string `json:"account"`
	LastChecked string `json:"lastChecked"`
}

func NewApp() *App {
	return &App{configPath: config.Path(), state: AppState{Status: "idle", Message: "输入账号开始连接"}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if settings, err := config.Load(a.configPath); err == nil {
		a.settings = settings
	}
	a.refresh()
	go a.monitor()
}

func (a *App) State() AppState { a.mu.Lock(); defer a.mu.Unlock(); return a.state }
func (a *App) Settings() config.Settings { a.mu.Lock(); defer a.mu.Unlock(); return a.settings }

func (a *App) SaveSettings(settings config.Settings) error {
	if settings.Role != "student" && settings.Role != "teacher" { return fmt.Errorf("invalid account role") }
	if settings.ISP == "" { settings.ISP = "cucc" }
	if err := config.Save(a.configPath, settings); err != nil { return err }
	a.mu.Lock(); a.settings = settings; a.mu.Unlock()
	return startup.Set(settings.AutoStart)
}

func (a *App) Login(username, password, role, isp string, remember bool) error {
	if username == "" || password == "" { return fmt.Errorf("请输入账号和密码") }
	info, err := network.Detect()
	if err != nil { a.setState(AppState{Status: "offline", Message: err.Error()}); return err }
	a.setState(AppState{Status: "connecting", Message: "正在连接校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal})
	client, err := portal.New("http://192.168.99.135", "1", net.ParseIP(info.IP))
	if err != nil { return err }
	creds := portal.Credentials{Username: username, Password: password, IPv4: info.IP, MAC: info.MAC, ISP: isp}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second); defer cancel()
	check, err := client.Check(ctx, creds)
	if err != nil { a.fail(err); return err }
	if check.Code != 0 { err = fmt.Errorf("账号检查失败：%s", check.Message); a.fail(err); return err }
	response, err := client.Login(ctx, creds)
	if err != nil { a.fail(err); return err }
	if response.Code != 0 { err = fmt.Errorf("登录失败：%s", response.Message); a.fail(err); return err }
	settings := config.Settings{Username: username, Password: password, Role: role, ISP: isp, Remember: remember, AutoStart: a.settings.AutoStart}
	if err := a.SaveSettings(settings); err != nil { return err }
	a.mu.Lock(); a.client, a.info = client, info; a.mu.Unlock()
	a.setState(AppState{Status: "connected", Message: "已连接校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal, Account: mask(username), LastChecked: time.Now().Format("15:04:05")})
	return nil
}

func (a *App) Logout() error {
	a.mu.Lock(); client, info, settings := a.client, a.info, a.settings; a.mu.Unlock()
	if client == nil { a.setState(AppState{Status: "idle", Message: "当前没有活动连接"}); return nil }
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second); defer cancel()
	_, err := client.Logout(ctx, portal.Credentials{Username: settings.Username, IPv4: info.IP, MAC: info.MAC, ISP: settings.ISP})
	a.mu.Lock(); a.client = nil; a.mu.Unlock()
	a.setState(AppState{Status: "idle", Message: "已断开校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal})
	return err
}

func (a *App) Refresh() AppState { a.refresh(); return a.State() }

func (a *App) refresh() {
	info, err := network.Detect()
	if err != nil { a.setState(AppState{Status: "offline", Message: err.Error()}); return }
	a.mu.Lock(); state := a.state; state.SSID, state.Interface, state.IP, state.MAC, state.Signal = info.SSID, info.Interface, info.IP, info.MAC, info.Signal; a.state = state; a.mu.Unlock()
}

func (a *App) monitor() {
	ticker := time.NewTicker(20 * time.Second); defer ticker.Stop()
	for { select { case <-a.ctx.Done(): return; case <-ticker.C: a.refresh() } }
}

func (a *App) setState(state AppState) { a.mu.Lock(); a.state = state; a.mu.Unlock(); if a.ctx != nil { runtime.EventsEmit(a.ctx, "state:changed", state) } }
func (a *App) fail(err error) { a.setState(AppState{Status: "error", Message: err.Error()}) }
func mask(value string) string { if len(value) <= 3 { return "***" }; return value[:2] + "***" + value[len(value)-1:] }

