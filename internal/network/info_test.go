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
