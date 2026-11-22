package africastalking

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nasoma/atcli/internal/money"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
)

type Error struct {
	Kind string
	Code int
}

func (e *Error) Error() string { return fmt.Sprintf("Africa's Talking %s (HTTP %d)", e.Kind, e.Code) }

type Client struct {
	base, user, key string
	http            *http.Client
}

func New(env, user, key string) (*Client, error) {
	base := "https://api.sandbox.africastalking.com"
	if env == "production" {
		base = "https://api.africastalking.com"
	} else if env != "sandbox" {
		return nil, fmt.Errorf("invalid environment")
	}
	return &Client{base, user, key, &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}
func (c *Client) request(ctx context.Context, method, path string, form url.Values, op string, out any) error {
	u := c.base + path
	var body io.Reader
	if method == http.MethodGet {
		u += "?" + form.Encode()
	} else {
		body = strings.NewReader(form.Encode())
	}
	req, e := http.NewRequestWithContext(ctx, method, u, body)
	if e != nil {
		return &Error{Kind: "request error"}
	}
	req.Header.Set("apiKey", c.key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if op != "" {
		req.Header.Set("Idempotency-Key", op)
	}
	res, e := c.http.Do(req)
	if e != nil {
		return &Error{Kind: "network failure; outcome may be unresolved"}
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		kind := "HTTP error"
		if res.StatusCode == 401 || res.StatusCode == 403 {
			kind = "authentication failed"
		}
		return &Error{kind, res.StatusCode}
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 1048577))
	if e != nil || len(b) > 1048576 {
		return &Error{Kind: "invalid or oversized response"}
	}
	if json.Unmarshal(b, out) != nil {
		return &Error{Kind: "malformed JSON response"}
	}
	return nil
}

type Balance struct {
	Currency string `json:"currency"`
	Minor    *int64 `json:"minor_units,omitempty"`
	Amount   string `json:"amount"`
}

func (c *Client) Balance(ctx context.Context) (Balance, error) {
	var r struct {
		UserData struct {
			Balance string `json:"balance"`
		} `json:"UserData"`
	}
	e := c.request(ctx, "GET", "/version1/user", url.Values{"username": {c.user}}, "", &r)
	if e != nil {
		return Balance{}, e
	}
	p := strings.Fields(r.UserData.Balance)
	if len(p) != 2 || !regexp.MustCompile(`^[A-Z]{3}$`).MatchString(p[0]) {
		return Balance{}, fmt.Errorf("invalid balance fields")
	}
	if !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`).MatchString(p[1]) {
		return Balance{}, fmt.Errorf("invalid balance amount")
	}
	amount := p[1]
	parts := strings.Split(amount, ".")
	exact := amount
	if len(parts) == 2 && len(parts[1]) > 2 {
		if strings.Trim(parts[1][2:], "0") != "" {
			return Balance{Currency: p[0], Amount: amount}, nil
		}
		exact = parts[0] + "." + parts[1][:2]
	}
	n, e := money.Parse(exact)
	if strings.Trim(strings.ReplaceAll(exact, ".", ""), "0") == "" {
		n = 0
		e = nil
	}
	if e != nil {
		return Balance{}, fmt.Errorf("invalid balance amount")
	}
	return Balance{p[0], &n, money.Format(n)}, nil
}

type Entry struct {
	Phone        string `json:"phoneNumber"`
	Amount       string `json:"amount"`
	RequestID    string `json:"requestId"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
}

func (c *Client) Send(ctx context.Context, phone, currency, amount, op string) (Entry, error) {
	rec, _ := json.Marshal([]map[string]string{{"phoneNumber": phone, "amount": currency + " " + amount}})
	var r struct {
		ErrorMessage string  `json:"errorMessage"`
		Responses    []Entry `json:"responses"`
	}
	e := c.request(ctx, "POST", "/version1/airtime/send", url.Values{"username": {c.user}, "recipients": {string(rec)}, "maxNumRetry": {"0"}}, op, &r)
	if e != nil {
		return Entry{}, e
	}
	if len(r.Responses) != 1 {
		return Entry{}, fmt.Errorf("provider returned no unique recipient result; reconcile in dashboard")
	}
	v := r.Responses[0]
	if v.Phone != phone || v.Status == "" {
		return Entry{}, fmt.Errorf("invalid airtime response fields; reconcile in dashboard")
	}
	if v.Status == "Sent" && v.RequestID == "" {
		return Entry{}, fmt.Errorf("accepted response missing request ID; reconcile in dashboard")
	}
	v.ErrorMessage = c.clean(v.ErrorMessage)
	v.RequestID = c.clean(v.RequestID)
	v.Status = c.clean(v.Status)
	return v, nil
}
func (c *Client) Status(ctx context.Context, id string) (string, error) {
	var r struct {
		Status string `json:"status"`
	}
	e := c.request(ctx, "GET", "/query/transaction/find", url.Values{"username": {c.user}, "transactionId": {id}}, "", &r)
	if e != nil {
		return "", e
	}
	if r.Status == "" {
		return "", fmt.Errorf("missing transaction status")
	}
	return c.clean(r.Status), nil
}
func Delivery(s string) string {
	switch strings.ToLower(s) {
	case "success":
		return "delivered"
	case "failed", "rejected":
		return "failed"
	case "sent", "pending", "queued":
		return "pending"
	default:
		return "unknown"
	}
}

func (c *Client) clean(s string) string {
	if c.key != "" {
		s = strings.ReplaceAll(s, c.key, "[redacted]")
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
