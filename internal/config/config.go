package config

import (
	"encoding/json"
	"fmt"
	"github.com/nasoma/atcli/internal/money"
	"github.com/zalando/go-keyring"
	"os"
	"path/filepath"
)

type Profile struct {
	Username string `json:"username"`
}
type Config struct {
	Environment string             `json:"environment"`
	Currency    string             `json:"default_currency"`
	Maximum     string             `json:"maximum"`
	Daily       string             `json:"daily_limit"`
	Profiles    map[string]Profile `json:"profiles"`
}

func Default() Config {
	return Config{"production", "KES", "1000", "0", map[string]Profile{"sandbox": {Username: "sandbox"}, "production": {}}}
}
func Dir() (string, error) {
	if v := os.Getenv("ATCLI_HOME"); v != "" {
		return v, nil
	}
	d, e := os.UserConfigDir()
	return filepath.Join(d, "atcli"), e
}
func Load() (Config, error) {
	c := Default()
	d, e := Dir()
	if e != nil {
		return c, e
	}
	b, e := os.ReadFile(filepath.Join(d, "config.json"))
	if os.IsNotExist(e) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	if e = json.Unmarshal(b, &c); e != nil {
		return c, fmt.Errorf("invalid config: %w", e)
	}
	return c, c.Validate()
}
func Exists() bool {
	d, e := Dir()
	if e != nil {
		return false
	}
	_, e = os.Stat(filepath.Join(d, "config.json"))
	return e == nil
}
func (c Config) Validate() error {
	if c.Environment != "sandbox" && c.Environment != "production" {
		return fmt.Errorf("environment must be sandbox or production")
	}
	if !money.Currency(c.Currency) {
		return fmt.Errorf("unsupported currency")
	}
	if _, e := money.Parse(c.Maximum); e != nil {
		return e
	}
	if c.Daily != "0" {
		if _, e := money.Parse(c.Daily); e != nil {
			return e
		}
	}
	if c.Profiles == nil {
		return fmt.Errorf("missing profiles")
	}
	return nil
}
func (c Config) Save() error {
	if e := c.Validate(); e != nil {
		return e
	}
	d, e := Dir()
	if e != nil {
		return e
	}
	if e = os.MkdirAll(d, 0700); e != nil {
		return e
	}
	b, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(d, "config-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(name, filepath.Join(d, "config.json"))
}
func (c Config) Credentials(env string) (string, string, error) {
	u := c.Profiles[env].Username
	if v := os.Getenv("AT_USERNAME"); v != "" {
		u = v
	}
	k := os.Getenv("AT_API_KEY")
	if k == "" {
		k, _ = keyring.Get("atcli/"+env, u)
	}
	if u == "" || k == "" {
		return "", "", fmt.Errorf("Africa's Talking credentials missing for %s; run atcli config init or set AT_USERNAME and AT_API_KEY", env)
	}
	if env == "sandbox" && u != "sandbox" {
		return "", "", fmt.Errorf("sandbox username must be sandbox")
	}
	return u, k, nil
}
func StoreKey(env, user, key string) error {
	if key == "" {
		return fmt.Errorf("API key is empty")
	}
	if e := keyring.Set("atcli/"+env, user, key); e != nil {
		return fmt.Errorf("secure credential store unavailable; use AT_API_KEY instead (no plaintext fallback)")
	}
	return nil
}
