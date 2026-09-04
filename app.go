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
	"net/url"
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
	sessionActive bool
	trafficMu     sync.Mutex
	trafficIn     uint64
	trafficOut    uint64
	trafficAt     time.Time
}

type AppState struct {
	Status       string   `json:"status"`
	Message      string   `json:"message"`
	SSID         string   `json:"ssid"`
	Interface    string   `json:"interface"`
	IP           string   `json:"ip"`
	MAC          string   `json:"mac"`
	Signal       string   `json:"signal"`
	Provider     string   `json:"provider"`
	Account      string   `json:"account"`
	LastChecked  string   `json:"lastChecked"`
	Networks     []string `json:"networks"`
	BytesIn4     int64    `json:"bytesIn4"`
	BytesOut4    int64    `json:"bytesOut4"`
	OnlineCount  int      `json:"onlineCount"`
	Terminals    []string `json:"terminals"`
	AuthCode     string   `json:"authCode"`
	AuthMessage  string   `json:"authMessage"`
	DialCode     string   `json:"dialCode"`
	DialMessage  string   `json:"dialMessage"`
	DownloadRate int64    `json:"downloadRate"`
	UploadRate   int64    `json:"uploadRate"`
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
	AssetURL    string `json:"assetUrl"`
}

// appVersion is replaced by the release workflow with the pushed tag.
// Keeping a development default makes local builds self-describing.
var appVersion = "0.1.0-dev"

func (a *App) About() AboutInfo {
	info := AboutInfo{Version: strings.TrimPrefix(appVersion, "v"), Project: "https://github.com/XtSilan/hbasstuNet", Issues: "https://github.com/XtSilan/hbasstuNet/issues"}
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
		Assets      []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(res.Body).Decode(&release); err != nil {
		return UpdateInfo{}, fmt.Errorf("解析更新信息失败：%w", err)
	}
	assetURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			assetURL = asset.URL
			break
		}
	}
	return UpdateInfo{Status: "检查完成", Version: release.TagName, Name: release.Name, Notes: release.Body, URL: release.HTMLURL, PublishedAt: release.PublishedAt, AssetURL: assetURL}, nil
}

func (a *App) InstallUpdate(assetURL string) error {
	parsed, err := url.Parse(assetURL)
	if err != nil || parsed.Scheme != "https" || (parsed.Host != "github.com" && parsed.Host != "objects.githubusercontent.com") {
		return fmt.Errorf("更新地址无效")
	}
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取程序路径失败：%w", err)
	}
	temporary := executable + ".update"
	file, err := os.Create(temporary)
	if err != nil {
		return fmt.Errorf("创建更新文件失败：%w", err)
	}
	defer file.Close()
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		os.Remove(temporary)
		return err
	}
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("User-Agent", "hbasstuNet/"+appVersion)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		os.Remove(temporary)
		return fmt.Errorf("下载更新失败：%w", err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		os.Remove(temporary)
		return fmt.Errorf("下载更新失败：GitHub 返回 HTTP %d", res.StatusCode)
	}
	if _, err := io.Copy(file, res.Body); err != nil {
		res.Body.Close()
		os.Remove(temporary)
		return fmt.Errorf("写入更新文件失败：%w", err)
	}
	res.Body.Close()
	if err := file.Close(); err != nil {
		os.Remove(temporary)
		return err
	}
	if err := launchUpdater(temporary, executable, os.Getpid()); err != nil {
		os.Remove(temporary)
		return err
	}
	log.Printf("update downloaded; restarting from %s", temporary)
	a.mu.Lock()
	a.allowClose = true
	a.mu.Unlock()
	runtime.Quit(a.ctx)
	return nil
}

func NewApp() *App {
	return &App{configPath: config.Path(), state: AppState{Status: "idle", Message: "输入账号开始连接", Networks: []string{}, Terminals: []string{}}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	initToast()
	if settings, err := config.Load(a.configPath); err == nil {
		a.settings = settings
		// The executable is portable. Refresh the Run entry on every launch so
		// moving the single-file app and opening it once repairs the old path.
		if settings.AutoStart {
			if err := startup.SyncCurrentPath(); err != nil {
				log.Printf("sync auto login path failed: %v", err)
			}
		}
	} else {
		log.Printf("load settings failed: %v", err)
	}
	log.Printf("application startup callback completed")
	go a.initialize()
	go a.monitor()
	go a.monitorTraffic()
	go a.startTray()
}

func (a *App) initialize() {
	// Discover nearby campus networks, then optionally authenticate with saved
	// credentials when automatic login is enabled.
	a.refresh()
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()
	if settings.AutoLogin && settings.Username != "" && settings.Password != "" {
		if err := a.Login(settings.Username, settings.Password, settings.Role, settings.Remember); err != nil {
			log.Printf("background automatic login failed: %v", err)
		}
	}
}

func (a *App) State() AppState { a.mu.Lock(); defer a.mu.Unlock(); return a.state }
func (a *App) Settings() config.Settings {
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()
	if enabled, err := startup.Enabled(); err == nil {
		settings.AutoStart = enabled
	}
	return settings
}

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
	if err := startup.Set(settings.AutoStart); err != nil {
		log.Printf("set auto start enabled=%t failed: %v", settings.AutoStart, err)
		return fmt.Errorf("更新开机自启动失败：%w", err)
	}
	log.Printf("auto start updated; enabled=%t", settings.AutoStart)
	log.Printf("settings saved; remember=%t autoLogin=%t autoStart=%t role=%s", settings.Remember, settings.AutoLogin, settings.AutoStart, settings.Role)
	return nil
}

func (a *App) Login(username, password, role string, remember bool) error {
	log.Printf("login requested; role=%s remember=%t", role, remember)
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
	creds := portal.Credentials{Username: username, Password: password, IPv4: info.IP, MAC: info.MAC}
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()
	status, statusErr := client.Status(ctx, creds)
	if statusErr == nil && status.ISP != "" {
		creds.ISP = status.ISP
	}
	if statusErr == nil && responseAuthenticated(status) && responseAccountMatches(status, username) {
		account := username
		if status.Online != nil && status.Online.Username != "" {
			account = status.Online.Username
		}
		a.setConnected(client, info, account, networks, status)
		return nil
	}
	if statusErr == nil && status.Online != nil && status.Online.Username != "" && !responseAccountMatches(status, username) {
		oldCredentials := creds
		if status.Online != nil && status.Online.Username != "" {
			oldCredentials.Username = status.Online.Username
		}
		if status.ISP != "" {
			oldCredentials.ISP = status.ISP
		}
		cleanupCtx, cleanupCancel := context.WithTimeout(a.ctx, 10*time.Second)
		if _, logoutErr := client.Logout(cleanupCtx, oldCredentials); logoutErr != nil {
			log.Printf("previous portal session cleanup failed: %v", logoutErr)
		}
		cleanupCancel()
	}
	check, err := client.Check(ctx, creds)
	if err != nil {
		a.failLogin(err, client, info, creds, networks)
		return err
	}
	if check.Code != 0 {
		err = responseError("账号检查失败", check)
		a.failLogin(err, client, info, creds, networks)
		return err
	}
	response, err := client.Login(ctx, creds)
	if err != nil {
		a.failLogin(err, client, info, creds, networks)
		return err
	}
	if !responseAuthenticated(response) {
		err = responseError("登录失败", response)
		a.failLogin(err, client, info, creds, networks)
		return err
	}
	log.Printf("portal login succeeded; ssid=%s ip=%s account=%s", info.SSID, info.IP, mask(username))
	a.mu.Lock()
	settings := a.settings
	a.mu.Unlock()
	settings.Username, settings.Password, settings.Role, settings.Remember = username, password, role, remember
	if response.ISP != "" {
		settings.ISP = response.ISP
	}
	if err := a.SaveSettings(settings); err != nil {
		return err
	}
	a.setConnected(client, info, username, networks, response)
	return nil
}

func responseAuthenticated(response portal.Response) bool {
	if response.Code != 0 || response.Online == nil {
		return false
	}
	if code := strings.ToUpper(strings.TrimSpace(response.AuthCode)); code != "" && !strings.HasPrefix(code, "OK:") {
		return false
	}
	if code := strings.ToUpper(strings.TrimSpace(response.DialCode)); strings.HasPrefix(code, "E") {
		return false
	}
	return true
}

func responseAccountMatches(response portal.Response, username string) bool {
	if response.Online == nil || strings.TrimSpace(response.Online.Username) == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(response.Online.Username), strings.TrimSpace(username))
}

func responseError(prefix string, response portal.Response) error {
	message := strings.TrimSpace(response.DialMessage)
	if message == "" {
		message = strings.TrimSpace(response.AuthMessage)
	}
	if message == "" {
		message = strings.TrimSpace(response.Message)
	}
	if message == "" {
		message = strings.TrimSpace(response.DialCode)
	}
	if message == "" {
		message = "校园网认证未成功"
	}
	return fmt.Errorf("%s：%s", prefix, message)
}

func providerName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "telecom", "ctcc", "电信":
		return "中国电信"
	case "cucc", "unicom", "联通":
		return "中国联通"
	case "cmcc", "mobile", "移动":
		return "中国移动"
	case "local", "校园网":
		return "校园网"
	default:
		return strings.TrimSpace(value)
	}
}

func (a *App) setConnected(client *portal.Client, info network.Info, username string, networks []string, response portal.Response) {
	a.mu.Lock()
	wasConnected := a.state.Status == "connected"
	a.client, a.info = client, info
	a.sessionActive = true
	if response.ISP != "" {
		a.settings.ISP = response.ISP
	}
	a.mu.Unlock()
	provider := providerName(response.ISP)
	message := "已连接校园网"
	if provider != "" {
		message = "已连接" + provider
	}
	state := AppState{Status: "connected", Message: message, SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal, Provider: provider, Account: username, Networks: networks, LastChecked: time.Now().Format("15:04:05"), AuthCode: response.AuthCode, AuthMessage: response.AuthMessage, DialCode: response.DialCode, DialMessage: response.DialMessage}
	if response.Online != nil {
		state.BytesIn4, state.BytesOut4, state.OnlineCount = response.Online.BytesIn4, response.Online.BytesOut4, 1
		state.Terminals = []string{response.Online.UserMAC + " · " + response.Online.UserIPv4}
	}
	a.setState(state)
	if !wasConnected {
		showToast("已连接校园网", info.SSID+" · "+username)
	}
}

func (a *App) Logout() error {
	a.mu.Lock()
	client, info, settings := a.client, a.info, a.settings
	wasConnected := a.state.Status == "connected" || client != nil
	a.mu.Unlock()
	var err error
	if client != nil {
		ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
		_, err = client.Logout(ctx, portal.Credentials{Username: settings.Username, IPv4: info.IP, MAC: info.MAC, ISP: settings.ISP})
		cancel()
	}
	if wifiErr := network.Disconnect(); wifiErr != nil && client != nil {
		if err == nil {
			err = wifiErr
		}
		log.Printf("Wi-Fi disconnect failed: %v", wifiErr)
	}
	a.mu.Lock()
	a.client = nil
	a.sessionActive = false
	a.mu.Unlock()
	a.setState(AppState{Status: "idle", Message: "已断开校园网", SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal})
	if wasConnected {
		showToast("已断开校园网", "校园网连接已断开")
	}
	return err
}

// failLogin logs out the portal session created during this login attempt and
// clears local session state before returning the authentication error. The
// portal identifies the session by client IPv4/MAC, so this also cleans up a
// partially authenticated account when credentials are rejected.
func (a *App) failLogin(err error, client *portal.Client, info network.Info, credentials portal.Credentials, networks []string) {
	if client != nil {
		base := a.ctx
		if base == nil {
			base = context.Background()
		}
		ctx, cancel := context.WithTimeout(base, 10*time.Second)
		if _, logoutErr := client.Logout(ctx, credentials); logoutErr != nil {
			log.Printf("cleanup portal session failed: %v", logoutErr)
		}
		cancel()
	}
	a.mu.Lock()
	a.client = nil
	a.sessionActive = false
	a.mu.Unlock()
	log.Printf("login failed; portal session cleared: %v", err)
	a.setState(AppState{Status: "error", Message: err.Error(), SSID: info.SSID, Interface: info.Interface, IP: info.IP, MAC: info.MAC, Signal: info.Signal, Networks: networks})
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
		a.mu.Lock()
		a.client = nil
		a.sessionActive = false
		a.mu.Unlock()
		a.setState(AppState{Status: "offline", Message: message, Networks: networks})
		return
	}
	a.mu.Lock()
	settings := a.settings
	state := a.state
	sessionActive := a.sessionActive
	a.mu.Unlock()
	state.SSID, state.Interface, state.IP, state.MAC, state.Signal, state.Networks = info.SSID, info.Interface, info.IP, info.MAC, info.Signal, networks
	if sessionActive && settings.Username != "" && settings.Password != "" {
		client, clientErr := portal.New("http://192.168.99.135", "1", net.ParseIP(info.IP))
		if clientErr == nil {
			ctx, cancel := context.WithTimeout(a.ctx, 8*time.Second)
			response, statusErr := client.Status(ctx, portal.Credentials{Username: settings.Username, Password: settings.Password, IPv4: info.IP, MAC: info.MAC, ISP: settings.ISP})
			cancel()
			if statusErr == nil && responseAuthenticated(response) {
				account := settings.Username
				if response.Online != nil && response.Online.Username != "" {
					account = response.Online.Username
				}
				a.setConnected(client, info, account, networks, response)
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
			sessionActive := a.sessionActive
			a.mu.Unlock()
			// AutoLogin controls startup behaviour; once a campus Wi-Fi is present,
			// a lost portal session is restored automatically when credentials exist.
			if sessionActive && settings.Username != "" && settings.Password != "" && state.Status == "offline" && strings.Contains(state.Message, "等待认证") {
				if err := a.Login(settings.Username, settings.Password, settings.Role, settings.Remember); err != nil {
					log.Printf("automatic re-authentication failed: %v", err)
				}
			}
		}
	}
}

func (a *App) monitorTraffic() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			iface := a.state.Interface
			current := a.state
			a.mu.Unlock()
			if iface == "" {
				continue
			}
			inBytes, outBytes, err := network.Traffic(iface)
			if err != nil {
				continue
			}
			now := time.Now()
			a.trafficMu.Lock()
			if !a.trafficAt.IsZero() {
				seconds := now.Sub(a.trafficAt).Seconds()
				if seconds > 0 {
					current.DownloadRate = int64(float64(maxDelta(inBytes, a.trafficIn)) / seconds)
					current.UploadRate = int64(float64(maxDelta(outBytes, a.trafficOut)) / seconds)
				}
			}
			a.trafficIn, a.trafficOut, a.trafficAt = inBytes, outBytes, now
			a.trafficMu.Unlock()
			if current.Status == "connected" {
				current.BytesIn4 = int64(inBytes)
				current.BytesOut4 = int64(outBytes)
				a.setState(current)
			}
		}
	}
}

func maxDelta(current, previous uint64) uint64 {
	if current < previous {
		return 0
	}
	return current - previous
}

func (a *App) setState(state AppState) {
	// JSON encodes nil slices as null. The Vue UI renders these collections
	// continuously while the monitor refreshes, so always expose arrays.
	if state.Networks == nil {
		state.Networks = []string{}
	}
	if state.Terminals == nil {
		state.Terminals = []string{}
	}
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
