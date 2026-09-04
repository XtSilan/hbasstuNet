package main

import (
	"testing"

	"hbasstuNet/internal/portal"
)

func TestResponseAuthenticated(t *testing.T) {
	tests := []struct {
		name string
		resp portal.Response
		want bool
	}{
		{name: "radius success", resp: portal.Response{Code: 0, AuthCode: "ok:radius", Online: &portal.Online{}}, want: true},
		{name: "auth error", resp: portal.Response{Code: 0, AuthCode: "E401", Online: &portal.Online{}}, want: false},
		{name: "dial error", resp: portal.Response{Code: 0, AuthCode: "ok:radius", DialCode: "E22904", Online: &portal.Online{}}, want: false},
		{name: "no online record", resp: portal.Response{Code: 0, AuthCode: "ok:radius"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := responseAuthenticated(tt.resp); got != tt.want {
				t.Fatalf("responseAuthenticated() = %t, want %t", got, tt.want)
			}
		})
	}
}
