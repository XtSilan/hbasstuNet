package network

import (
	"fmt"
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
	cmd := exec.Command("netsh", "wlan", "show", "networks", "mode=bssid")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("scan Wi-Fi networks: %w", err)
	}
	return parseNetworks(string(out)), nil
}

func Connect(ssid string) (Info, error) {
	if !campusSSID.MatchString(ssid) {
		return Info{}, fmt.Errorf("unsupported campus network: %s", ssid)
	}
	if info, err := Detect(); err == nil && strings.EqualFold(info.SSID, ssid) {
		return info, nil
	}
	if err := addOpenProfile(ssid); err != nil {
		return Info{}, fmt.Errorf("prepare Wi-Fi %s: %w", ssid, err)
	}
	cmd := exec.Command("netsh", "wlan", "connect", "name="+ssid, "ssid="+ssid)
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		return Info{}, fmt.Errorf("connect Wi-Fi %s: %w", ssid, err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if info, err := Detect(); err == nil && strings.EqualFold(info.SSID, ssid) {
			return info, nil
		}
		time.Sleep(500 * time.Millisecond)
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
	cmd := exec.Command("netsh", "wlan", "add", "profile", "filename="+path, "user=current")
	hideWindow(cmd)
	return cmd.Run()
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
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	hideWindow(cmd)
	if out, err := cmd.Output(); err == nil {
		return parseNetsh(string(out))
	}
	return detectInterfaces()
}

func parseNetsh(output string) (Info, error) {
	var info Info
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name":
			info.Interface = strings.TrimSpace(value)
		case "ssid":
			info.SSID = strings.TrimSpace(value)
		case "signal":
			info.Signal = strings.TrimSpace(value)
		}
	}
	if info.SSID == "" {
		return info, fmt.Errorf("no Wi-Fi connection")
	}
	info.Ready = campusSSID.MatchString(info.SSID)
	if info.Interface != "" {
		if iface, err := net.InterfaceByName(info.Interface); err == nil {
			info.MAC = iface.HardwareAddr.String()
			if addrs, err := iface.Addrs(); err == nil {
				for _, addr := range addrs {
					ip, _, e := net.ParseCIDR(addr.String())
					if e == nil && ip.To4() != nil {
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
