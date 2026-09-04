package portal

import (
	"bytes"
	"encoding/json"
	"strconv"
)

type Response struct {
	Code        int     `json:"code"`
	Message     string  `json:"msg"`
	AuthCode    string  `json:"authCode"`
	AuthMessage string  `json:"authMsg"`
	DialCode    string  `json:"dialCode"`
	DialMessage string  `json:"dialMsg"`
	EnableDial  bool    `json:"enableDial"`
	ISP         string  `json:"isp"`
	Online      *Online `json:"online"`
}

type Online struct {
	Username  string `json:"Username"`
	UserIPv4  string `json:"UserIpv4"`
	UserMAC   string `json:"UserMac"`
	SessionID string `json:"SessionId"`
	BytesIn4  int64  `json:"BytesIn4"`
	BytesOut4 int64  `json:"BytesOut4"`
	BytesIn6  int64  `json:"BytesIn6"`
	BytesOut6 int64  `json:"BytesOut6"`
	AddTime   string `json:"AddTime"`
}

// UnmarshalJSON accepts both numeric and quoted byte counters. The portal has
// returned both forms across firmware versions.
func (o *Online) UnmarshalJSON(data []byte) error {
	var raw struct {
		Username  string          `json:"Username"`
		UserIPv4  string          `json:"UserIpv4"`
		UserMAC   string          `json:"UserMac"`
		SessionID string          `json:"SessionId"`
		BytesIn4  json.RawMessage `json:"BytesIn4"`
		BytesOut4 json.RawMessage `json:"BytesOut4"`
		BytesIn6  json.RawMessage `json:"BytesIn6"`
		BytesOut6 json.RawMessage `json:"BytesOut6"`
		AddTime   string          `json:"AddTime"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	o.Username, o.UserIPv4, o.UserMAC, o.SessionID, o.AddTime = raw.Username, raw.UserIPv4, raw.UserMAC, raw.SessionID, raw.AddTime
	var err error
	if o.BytesIn4, err = parseCounter(raw.BytesIn4); err != nil {
		return err
	}
	if o.BytesOut4, err = parseCounter(raw.BytesOut4); err != nil {
		return err
	}
	if o.BytesIn6, err = parseCounter(raw.BytesIn6); err != nil {
		return err
	}
	if o.BytesOut6, err = parseCounter(raw.BytesOut6); err != nil {
		return err
	}
	return nil
}

func parseCounter(raw json.RawMessage) (int64, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, err
		}
		if value == "" {
			return 0, nil
		}
		return strconv.ParseInt(value, 10, 64)
	}
	var value int64
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, err
	}
	return value, nil
}
