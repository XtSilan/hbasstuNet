package network

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

type Info struct {
	SSID      string `json:"ssid"`
	Interface string `json:"interface"`
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	Signal    string `json:"signal"`
	Ready     bool   `json:"ready"`
}

var campusSSID = regexp.MustCompile(`(?i)^(student|teacher|tercher)-xyw$`)
var listedSSID = regexp.MustCompile(`(?im)^\s*SSID\s+\d+\s*:\s*(.+?)\s*$`)

func ScanCampus() ([]string, error) {
	out, err := runNetsh("wlan", "show", "networks", "mode=bssid")
	if err != nil {
		return nil, fmt.Errorf("扫描附近 Wi-Fi 失败：%s", commandMessage(err, out))
	}
	return parseNetworks(out), nil
}

func Connect(ssid string) (Info, error) {
	if !campusSSID.MatchString(ssid) {
		return Info{}, fmt.Errorf("unsupported campus network: %s", ssid)
	}
	initial, initialErr := Detect()
	alreadyAssociated := strings.EqualFold(initial.SSID, ssid)
	if initialErr == nil && alreadyAssociated {
		return initial, nil
	}
	initialIP := initial.IP
	if !hasProfile(ssid) {
		if err := addOpenProfile(ssid); err != nil {
			return Info{}, fmt.Errorf("创建 Wi-Fi 配置 %s 失败：%w", ssid, err)
		}
	}
	out, err := runNetsh("wlan", "connect", "name="+ssid, "ssid="+ssid)
	if err != nil {
		return Info{}, fmt.Errorf("连接 Wi-Fi %s 失败：%s", ssid, commandMessage(err, out))
	}
	// Windows may report the new SSID before DHCP has assigned an address.
	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		info, detectErr := Detect()
		if strings.EqualFold(info.SSID, ssid) {
			if detectErr == nil {
				if alreadyAssociated || initialIP == "" || info.IP != initialIP {
					return info, nil
				}
				lastErr = fmt.Errorf("正在等待校园网分配 IPv4 地址")
			} else {
				lastErr = detectErr
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	if lastErr != nil {
		return Info{}, fmt.Errorf("连接 %s 超时：%v", ssid, lastErr)
	}
	return Info{}, fmt.Errorf("连接 %s 超时", ssid)
}

func addOpenProfile(ssid string) error {
	profile := fmt.Sprintf(`<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
  <name>%s</name>
  <SSIDConfig><SSID><name>%s</name></SSID></SSIDConfig>
  <connectionType>ESS</connectionType>
  <connectionMode>manual</connectionMode>
  <MSM><security><authEncryption><authentication>open</authentication><encryption>none</encryption><useOneX>false</useOneX></authEncryption></security></MSM>
</WLANProfile>`, ssid, ssid)
	file, err := os.CreateTemp("", "hbasstunet-wifi-*.xml")
	if err != nil {
		return err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(profile); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	out, err := runNetsh("wlan", "add", "profile", "filename="+path, "user=current")
	if err != nil {
		return fmt.Errorf("%s", commandMessage(err, out))
	}
	return nil
}

func hasProfile(ssid string) bool {
	_, err := runNetsh("wlan", "show", "profile", "name="+ssid)
	return err == nil
}

func parseNetworks(output string) []string {
	seen := make(map[string]bool)
	var networks []string
	for _, match := range listedSSID.FindAllStringSubmatch(output, -1) {
		ssid := strings.TrimSpace(match[1])
		if campusSSID.MatchString(ssid) && !seen[strings.ToLower(ssid)] {
			seen[strings.ToLower(ssid)] = true
			networks = append(networks, ssid)
		}
	}
	return networks
}

func Detect() (Info, error) {
	if out, err := runNetsh("wlan", "show", "interfaces"); err == nil {
		return parseNetsh(out)
	}
	return detectInterfaces()
}

func runNetsh(args ...string) (string, error) {
	cmd := exec.Command("netsh", args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		log.Printf("netsh %s failed: %v; output=%q", strings.Join(args, " "), err, text)
	}
	return text, err
}

func commandMessage(err error, output string) string {
	message := strings.TrimSpace(output)
	if message == "" {
		return err.Error()
	}
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.ReplaceAll(message, "\n", " ")
	return strings.Join(strings.Fields(message), " ")
}

func parseNetsh(output string) (Info, error) {
	info := parseNetshFields(output)
	if info.SSID == "" {
		return info, fmt.Errorf("no Wi-Fi connection")
	}
	info.Ready = campusSSID.MatchString(info.SSID)
	if info.Interface != "" {
		if iface, err := net.InterfaceByName(info.Interface); err == nil {
			if len(iface.HardwareAddr) > 0 {
				info.MAC = iface.HardwareAddr.String()
			}
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					ip, _, e := net.ParseCIDR(addr.String())
					if e == nil && ip.To4() != nil && !ip.IsLinkLocalUnicast() {
						info.IP = ip.To4().String()
						break
					}
				}
			}
		}
	}
	if !info.Ready {
		return info, fmt.Errorf("not a campus network: %s", info.SSID)
	}
	if info.IP == "" || info.MAC == "" {
		return info, fmt.Errorf("campus Wi-Fi has no IPv4 address")
	}
	return info, nil
}

func parseNetshFields(output string) Info {
	var info Info
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name", "名称":
			info.Interface = strings.TrimSpace(value)
		case "ssid":
			info.SSID = strings.TrimSpace(value)
		case "signal", "信号":
			info.Signal = strings.TrimSpace(value)
		case "physical address", "物理地址":
			info.MAC = strings.TrimSpace(value)
		}
	}
	return info
}

func detectInterfaces() (Info, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return Info{}, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, e := net.ParseCIDR(addr.String())
			if e == nil && ip.To4() != nil {
				return Info{Interface: iface.Name, IP: ip.To4().String(), MAC: iface.HardwareAddr.String(), Ready: true}, nil
			}
		}
	}
	return Info{}, fmt.Errorf("no active IPv4 interface")
}
