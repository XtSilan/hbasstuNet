package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Credentials struct {
	Username string
	Password string
	IPv4     string
	MAC      string
	ISP      string
}

type Client struct {
	baseURL string
	nasID   string
	csrf    string
	http    *http.Client
}

func New(baseURL, nasID string, sourceIP net.IP) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 8 * time.Second}
	if sourceIP != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: sourceIP}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), nasID: nasID, http: &http.Client{
		Jar: jar, Timeout: 15 * time.Second,
		Transport: &http.Transport{DialContext: dialer.DialContext},
	}}, nil
}

func (c *Client) Check(ctx context.Context, credentials Credentials) (Response, error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return Response{}, err
	}
	return c.post(ctx, "/api/account/check", credentials)
}

func (c *Client) Login(ctx context.Context, credentials Credentials) (Response, error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return Response{}, err
	}
	return c.post(ctx, "/api/account/login", credentials)
}

func (c *Client) Status(ctx context.Context, credentials Credentials) (Response, error) {
	return c.get(ctx, "/api/account/status?"+c.values(credentials).Encode(), false)
}

func (c *Client) Logout(ctx context.Context, credentials Credentials) (Response, error) {
	if err := c.ensureCSRF(ctx); err != nil {
		return Response{}, err
	}
	return c.get(ctx, "/api/account/logout?"+c.values(credentials).Encode(), true)
}

func (c *Client) ensureCSRF(ctx context.Context) error {
	if c.csrf != "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/csrf-token", nil)
	if err != nil {
		return err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("get CSRF token: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("get CSRF token: HTTP %d", res.StatusCode)
	}
	var body struct {
		Token string `json:"csrf_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return fmt.Errorf("decode CSRF token: %w", err)
	}
	if body.Token == "" {
		return fmt.Errorf("empty CSRF token")
	}
	c.csrf = body.Token
	return nil
}

func (c *Client) post(ctx context.Context, path string, credentials Credentials) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, strings.NewReader(c.values(credentials).Encode()))
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.ajax(req)
	return c.do(req)
}

func (c *Client) get(ctx context.Context, path string, csrf bool) (Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return Response{}, err
	}
	if csrf {
		c.ajax(req)
	}
	return c.do(req)
}

func (c *Client) do(req *http.Request) (Response, error) {
	res, err := c.http.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Response{}, fmt.Errorf("portal returned HTTP %d", res.StatusCode)
	}
	var result Response
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return Response{}, fmt.Errorf("decode portal response: %w", err)
	}
	return result, nil
}

func (c *Client) ajax(req *http.Request) {
	req.Header.Set("X-CSRF-Token", c.csrf)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func (c *Client) values(credentials Credentials) url.Values {
	v := url.Values{}
	v.Set("nasId", c.nasID)
	v.Set("userIpv4", credentials.IPv4)
	v.Set("userMac", credentials.MAC)
	if credentials.Username != "" {
		v.Set("username", credentials.Username)
	}
	if credentials.Password != "" {
		v.Set("password", credentials.Password)
	}
	if credentials.ISP != "" {
		v.Set("isp", credentials.ISP)
	}
	return v
}
