package cmd

import (
	"bytes"
	"testing"
)

func TestArguments(t *testing.T) {
	for _, args := range [][]string{{"airtime", "send"}, {"airtime", "send", "0712345678", "0", "--dry-run"}, {"airtime", "send", "bad", "100", "--dry-run"}, {"history", "--limit", "-1"}, {"config", "set", "unknown", "value"}, {"balance", "--env", "bad"}} {
		t.Run(stringJoin(args), func(t *testing.T) {
			t.Setenv("ATCLI_HOME", t.TempDir())
			r := Root()
			r.SetArgs(args)
			r.SetOut(&bytes.Buffer{})
			if e := r.Execute(); e == nil {
				t.Fatal("expected error")
			}
		})
	}
}
func stringJoin(v []string) string {
	var s string
	for _, a := range v {
		s += a + " "
	}
	return s
}
