package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"hbasstuNet/internal/config"
	"hbasstuNet/internal/network"
	"hbasstuNet/internal/portal"
	"hbasstuNet/internal/startup"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx           context.Context
	mu            sync.Mutex
	settings      config.Settings
	configPath    string
	state         AppState
	client        *portal.Client
	info          network.Info
	allowClose    bool
	frontendReady bool
}

type AppState struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	SSID        string   `json:"ssid"`
	Interface   string   `json:"interface"`
	IP          string   `json:"ip"`
	MAC         string   `json:"mac"`
	Signal      string   `json:"signal"`
	Account     string   `json:"account"`
	LastChecked string   `json:"lastChecked"`
	Networks    []string `json:"networks"`
}

type AboutInfo struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
	Project string `json:"project"`
	Issues  string `json:"issues"`
}

type UpdateInfo struct {
	Status      string `json:"status"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	Notes       string `json:"notes"`
	URL         string `json:"url"`
	PublishedAt string `json:"publishedAt"`
}

const appVersion = "0.1.0"

func (a *App) About() AboutInfo {
	info := AboutInfo{Version: appVersion, Project: "https://github.com/XtSilan/hbasstuNet", Issues: "https://github.com/XtSilan/hbasstuNet/issues"}
	if executable, err := os.Executable(); err == nil {
		if file, err := os.Open(executable); err == nil {
			hash := sha256.New()
			if _, err := io.Copy(hash, file); err == nil {
				info.SHA256 = hex.EncodeToString(hash.Sum(nil))
			}
			file.Close()
		}
	}
	return info
}

func (a *App) CheckUpdate() (UpdateInfo, error) {
	ctx, cancel := context.WithTimeout(a.ctx, 12*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/XtSilan/hbasstuNet/releases/latest", nil)
	if err != nil {
		return UpdateInfo{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "hbasstuNet/"+appVersion)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateInfo{}, fmt.Errorf("检查更新失败：%w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return UpdateInfo{Status: "暂无发布版本"}, nil
	}
	if res.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("检查更新失败：GitHub 返回 HTTP %d", res.StatusCode)
	}
	var release struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		Body        string `json:"body"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		return UpdateInfo{}, fmt.Errorf("解析更新信息失败：%w", err)
	}
	return UpdateInfo{Status: "检查完成", Version: release.TagName, Name: release.Name, Notes: release.Body, URL: release.HTMLURL, PublishedAt: release.PublishedAt}, nil
}

func NewApp() *App {
	return &App{configPath: config.Path(), state: AppState{Status: "idle", Message: "输入账号开始连接"}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if settings, err := config.Load(a.configPath); err == nil {
		a.settings = settings
	} else {
		log.Printf("load settings failed: %v", err)
	}
	log.Printf("application startup callback completed")
	go a.initialize()
	go a.monitor()
	go a.startTray()
}

func (a *App) initialize() {
	// Keep Wails responsive while network discovery and optional auto-login run.
	a.refresh()
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()
	if settings.AutoLogin && settings.Username != "" && settings.Password != "" {
		if err := a.Login(settings.Username, settings.Password, settings.Role, settings.ISP, settings.Remember); err != nil {
			log.Printf("automatic login failed: %v", err)
		}
	}
}

func (a *App) State() AppState           { a.mu.Lock(); defer a.mu.Unlock(); return a.state }
func (a *App) Settings() config.Settings { a.mu.Lock(); defer a.mu.Unlock(); return a.settings }

func (a *App) MarkFrontendReady() {
	a.mu.Lock()
	a.frontendReady = true
	a.mu.Unlock()
}

func (a *App) SaveSettings(settings config.Settings) error {
	if settings.Role != "student" && settings.Role != "teacher" {
		return fmt.Errorf("invalid account role")
	}
	if settings.ISP == "" {
		settings.ISP = "cucc"
	}
	if err := config.Save(a.configPath, settings); err != nil {
		log.Printf("save settings failed: %v", err)
		return err
	}
	a.mu.Lock()
	a.settings = settings
	a.mu.Unlock()
	if err := startup.Set(settings.AutoLogin); err != nil {
		log.Printf("set auto login enabled=%t failed: %v", settings.AutoLogin, err)
		return fmt.Errorf("更新自动登录失败：%w", err)
	}
	log.Printf("auto login updated; enabled=%t", settings.AutoLogin)
	log.Printf("settings saved; remember=%t autoLogin=%t role=%s", settings.Remember, settings.AutoLogin, settings.Role)
	return nil
}

func (a *App) Login(username, password, role, isp string, remember bool) error {
	log.Printf("login requested; role=%s isp=%s remember=%t", role, isp, remember)
	if username == "" || password == "" {
		return fmt.Errorf("请输入账号和密码")
	}
	a.mu.Lock()
	networks := append([]string(nil), a.state.Networks...)
	a.mu.Unlock()
	ssid := networkForRole(networks, role)
	if ssid == "" {
		err := fmt.Errorf("附近没有发现对应的校园网络")
		a.setState(AppState{Status: "offline", Message: err.Error(), Networks: networks})
		return err
	}
	a.setState(AppState{Status: "connecting", Message: "正在连接 " + ssid, Networks: networks})
	info, err := network.Connect(ssid)
	if err != nil {
		log.Printf("Wi-Fi connection failed; ssid=%s error=%v", ssid, err)
		a.setState(AppState{Status: "offline", Message: err.Error(), Networks: networks})
		return err
	}
	a.setState(AppState{Status: "connecting", Message: "正在认证校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal, Networks: networks})
	client, err := portal.New("http://192.168.99.135", "1", net.ParseIP(info.IP))
	if err != nil {
		return err
	}
	creds := portal.Credentials{Username: username, Password: password, IPv4: info.IP, MAC: info.MAC, ISP: isp}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	status, statusErr := client.Status(ctx, creds)
	if statusErr == nil && status.Code == 0 {
		account := username
		if status.Online != nil && status.Online.Username != "" {
			account = status.Online.Username
		}
		a.setConnected(client, info, account, networks)
		return nil
	}
	check, err := client.Check(ctx, creds)
	if err != nil {
		a.fail(err)
		return err
	}
	if check.Code != 0 {
		err = fmt.Errorf("账号检查失败：%s", check.Message)
		a.fail(err)
		return err
	}
	response, err := client.Login(ctx, creds)
	if err != nil {
		a.fail(err)
		return err
	}
	if response.Code != 0 {
		err = fmt.Errorf("登录失败：%s", response.Message)
		a.fail(err)
		return err
	}
	log.Printf("portal login succeeded; ssid=%s ip=%s account=%s", info.SSID, info.IP, mask(username))
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()
	settings.Username, settings.Password, settings.Role, settings.ISP, settings.Remember = username, password, role, isp, remember
	if err := a.SaveSettings(settings); err != nil {
		return err
	}
	a.setConnected(client, info, username, networks)
	return nil
}

func (a *App) setConnected(client *portal.Client, info network.Info, username string, networks []string) {
	a.mu.Lock()
	a.client, a.info = client, info
	a.mu.Unlock()
	a.setState(AppState{Status: "connected", Message: "已连接校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal, Account: username, Networks: networks, LastChecked: time.Now().Format("15:04:05")})
}

func (a *App) Logout() error {
	a.mu.Lock()
	client, info, settings := a.client, a.info, a.settings
	a.mu.Unlock()
	if client == nil {
		a.setState(AppState{Status: "idle", Message: "当前没有活动连接"})
		return nil
	}
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	_, err := client.Logout(ctx, portal.Credentials{Username: settings.Username, IPv4: info.IP, MAC: info.MAC, ISP: settings.ISP})
	a.mu.Lock()
	a.client = nil
	a.mu.Unlock()
	a.setState(AppState{Status: "idle", Message: "已断开校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal})
	return err
}

func (a *App) Refresh() AppState { a.refresh(); return a.State() }

func (a *App) refresh() {
	networks, scanErr := network.ScanCampus()
	info, err := network.Detect()
	if err != nil {
		message := "未连接到校园网"
		if scanErr != nil {
			message = "无线网络扫描暂不可用"
		}
		a.setState(AppState{Status: "offline", Message: message, Networks: networks})
		return
	}
	a.mu.Lock()
	settings := a.settings
	state := a.state
	a.mu.Unlock()
	state.SSID, state.Interface, state.IP, state.MAC, state.Signal, state.Networks = info.SSID, info.Interface, info.IP, info.MAC, info.Signal, networks
	if settings.Username != "" && settings.Password != "" {
		client, clientErr := portal.New("http://192.168.99.135", "1", net.ParseIP(info.IP))
		if clientErr == nil {
			ctx, cancel := context.WithTimeout(a.ctx, 8*time.Second)
			response, statusErr := client.Status(ctx, portal.Credentials{Username: settings.Username, Password: settings.Password, IPv4: info.IP, MAC: info.MAC, ISP: settings.ISP})
			cancel()
			if statusErr == nil && response.Code == 0 {
				account := settings.Username
				if response.Online != nil && response.Online.Username != "" {
					account = response.Online.Username
				}
				a.setConnected(client, info, account, networks)
				return
			}
		}
	}
	state.Status = "offline"
	state.Message = "已连接校园网，等待认证"
	a.setState(state)
}

func (a *App) monitor() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refresh()
			a.mu.Lock()
			settings := a.settings
			state := a.state
			a.mu.Unlock()
			if settings.AutoLogin && settings.Username != "" && settings.Password != "" && state.Status == "offline" && strings.Contains(state.Message, "等待认证") {
				if err := a.Login(settings.Username, settings.Password, settings.Role, settings.ISP, settings.Remember); err != nil {
					log.Printf("automatic re-authentication failed: %v", err)
				}
			}
		}
	}
}

func (a *App) setState(state AppState) {
	a.mu.Lock()
	a.state = state
	a.mu.Unlock()
	a.updateTray(state)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "state:changed", state)
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	a.mu.Lock()
	if a.allowClose {
		a.mu.Unlock()
		return false
	}
	if !a.frontendReady {
		a.mu.Unlock()
		return false
	}
	settings := a.settings
	a.mu.Unlock()
	if settings.SkipExitPrompt {
		if settings.ExitBehavior == "exit" {
			return false
		}
		runtime.WindowHide(ctx)
		return true
	}
	if a.ctx != nil {
		go runtime.EventsEmit(a.ctx, "close:requested")
	}
	return true
}

func (a *App) CloseToTray(dontAsk bool) error {
	if err := a.updateExitPreference("tray", dontAsk); err != nil {
		return err
	}
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
	return nil
}

func (a *App) ExitApp(dontAsk bool) error {
	if err := a.updateExitPreference("exit", dontAsk); err != nil {
		return err
	}
	a.mu.Lock()
	a.allowClose = true
	a.mu.Unlock()
	a.stopTray()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
	return nil
}

func (a *App) updateExitPreference(behavior string, dontAsk bool) error {
	a.mu.Lock()
	settings := a.settings
	settings.ExitBehavior, settings.SkipExitPrompt = behavior, dontAsk
	a.mu.Unlock()
	return a.SaveSettings(settings)
}
func (a *App) fail(err error) {
	log.Printf("operation failed: %v", err)
	a.setState(AppState{Status: "error", Message: err.Error()})
}
func mask(value string) string {
	if len(value) <= 3 {
		return "***"
	}
	return value[:2] + "***" + value[len(value)-1:]
}

func networkForRole(networks []string, role string) string {
	for _, ssid := range networks {
		isStudent := strings.HasPrefix(strings.ToLower(ssid), "student-")
		if (role == "student" && isStudent) || (role == "teacher" && !isStudent) {
			return ssid
		}
	}
	return ""
}
