package portal

type Response struct {
	Code        int     `json:"code"`
	Message     string  `json:"msg"`
	AuthCode    string  `json:"authCode"`
	AuthMessage string  `json:"authMsg"`
	DialCode    string  `json:"dialCode"`
	DialMessage string  `json:"dialMsg"`
	EnableDial  bool    `json:"enableDial"`
	Online      *Online `json:"online"`
}

type Online struct {
	Username  string `json:"Username"`
	UserIPv4  string `json:"UserIpv4"`
	UserMAC   string `json:"UserMac"`
	SessionID string `json:"SessionId"`
}
