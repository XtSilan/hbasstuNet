package network

import (
	"reflect"
	"testing"
)

func TestParseNetworksFiltersCampusSSIDs(t *testing.T) {
	output := `SSID 1 : Home
    Network type            : Infrastructure
SSID 2 : Student-XYW
SSID 3 : Tercher-XYW
SSID 4 : Student-XYW
`
	want := []string{"Student-XYW", "Tercher-XYW"}
	if got := parseNetworks(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNetworks() = %#v, want %#v", got, want)
	}
}

func TestParseNetshSupportsChineseWindows(t *testing.T) {
	output := `系统上有 1 个接口:

    名称                   : WLAN
    物理地址               : 94:b6:09:2a:5f:a8
    状态                   : 已连接
    SSID                   : Student-XYW
    信号                   : 95%
`
	info := parseNetshFields(output)
	if info.Interface != "WLAN" || info.SSID != "Student-XYW" || info.Signal != "95%" {
		t.Fatalf("parseNetshFields() = %+v", info)
	}
	if info.MAC != "94:b6:09:2a:5f:a8" {
		t.Fatalf("parseNetshFields() MAC = %q", info.MAC)
	}
}
