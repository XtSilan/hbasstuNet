package network

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
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

func Detect() (Info, error) {
	if out, err := exec.Command("netsh", "wlan", "show", "interfaces").Output(); err == nil {
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
