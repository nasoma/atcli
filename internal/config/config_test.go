package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Setenv("ATCLI_HOME", t.TempDir())
	c, e := Load()
	if e != nil || c.Environment != "production" {
		t.Fatal(c, e)
	}
	if e = c.Save(); e != nil {
		t.Fatal(e)
	}
	d, _ := Dir()
	st, e := os.Stat(filepath.Join(d, "config.json"))
	if e != nil || st.Mode().Perm() != 0600 {
		t.Fatal("insecure config permissions", e)
	}
	t.Setenv("AT_USERNAME", "sandbox")
	t.Setenv("AT_API_KEY", "private-test-key")
	u, k, e := c.Credentials("sandbox")
	if e != nil || u != "sandbox" || k != "private-test-key" {
		t.Fatal("environment override failed")
	}
	c.Environment = "wrong"
	if e = c.Save(); e == nil {
		t.Fatal("bad environment saved")
	}
}
