package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nasoma/atcli/internal/africastalking"
	"github.com/nasoma/atcli/internal/config"
	"github.com/nasoma/atcli/internal/money"
	"github.com/nasoma/atcli/internal/service"
	"github.com/nasoma/atcli/internal/storage"
	"github.com/nasoma/atcli/internal/tui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"text/tabwriter"
)

var Version = "dev"

func buildVersion() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}

func Interactive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}
func Open(env string, auth bool) (*service.Service, error) {
	c, e := config.Load()
	if e != nil {
		return nil, e
	}
	if env == "" {
		env = os.Getenv("AT_ENVIRONMENT")
	}
	if env == "" {
		env = c.Environment
	}
	if env != "sandbox" && env != "production" {
		return nil, fmt.Errorf("environment must be sandbox or production")
	}
	u := c.Profiles[env].Username
	if v := os.Getenv("AT_USERNAME"); v != "" {
		u = v
	}
	var api *africastalking.Client
	if auth {
		var k string
		u, k, e = c.Credentials(env)
		if e != nil {
			return nil, e
		}
		api, e = africastalking.New(env, u, k)
		if e != nil {
			return nil, e
		}
	}
	d, e := config.Dir()
	if e != nil {
		return nil, e
	}
	st, e := storage.Open(filepath.Join(d, "history.sqlite"))
	if e != nil {
		return nil, e
	}
	return &service.Service{Config: c, Env: env, User: u, API: api, Store: st}, nil
}
func output(v any) error { return json.NewEncoder(os.Stdout).Encode(v) }
func Root() *cobra.Command {
	var env string
	var jsonMode bool
	root := &cobra.Command{Use: "atcli", Short: "Africa's Talking airtime manager", Version: buildVersion(), SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringVar(&env, "env", "", "sandbox or production (defaults to config)")
	root.PersistentFlags().BoolVar(&jsonMode, "json", false, "machine-readable output; purchases also require --yes")
	launch := func(cmd *cobra.Command, args []string) error {
		if jsonMode {
			return fmt.Errorf("select a command such as balance or history when using --json")
		}
		if !Interactive() {
			return cmd.Help()
		}
		return tui.Run(env)
	}
	root.RunE = launch
	root.AddCommand(&cobra.Command{Use: "tui", Short: "Open the interactive terminal interface", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if jsonMode {
			return fmt.Errorf("TUI is incompatible with --json")
		}
		if !Interactive() {
			return fmt.Errorf("TUI requires an interactive terminal")
		}
		return tui.Run(env)
	}})
	root.AddCommand(&cobra.Command{Use: "version", Short: "Print the build version", Args: cobra.NoArgs, Run: func(cmd *cobra.Command, args []string) { fmt.Fprintln(cmd.OutOrStdout(), buildVersion()) }})
	root.AddCommand(&cobra.Command{Use: "balance", Short: "Retrieve the application balance without spending", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		s, e := Open(env, true)
		if e != nil {
			return e
		}
		defer s.Store.DB.Close()
		b, e := s.API.Balance(cmd.Context())
		if e != nil {
			return e
		}
		if jsonMode {
			return output(b)
		}
		fmt.Printf("Africa's Talking Account\nEnvironment: %s\nUsername:    %s\nBalance:     %s %s\n", s.Env, s.User, b.Currency, b.Amount)
		return nil
	}})
	air := &cobra.Command{Use: "airtime", Short: "Send and reconcile airtime"}
	var currency string
	var dry, yes bool
	send := &cobra.Command{Use: "send PHONE AMOUNT", Short: "Validate, confirm and send one airtime purchase", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		s, e := Open(env, !dry)
		if e != nil {
			return e
		}
		defer s.Store.DB.Close()
		cur := currency
		if cur == "" {
			cur = s.Config.Currency
		}
		r, e := s.Preview(args[0], args[1], cur)
		if e != nil {
			return e
		}
		if dry {
			r.Submission = "not_submitted"
			r.Status = "not_submitted"
			if jsonMode {
				return output(r)
			}
			fmt.Println(summary(r) + "\nDry run: no request submitted.")
			return nil
		}
		if !yes {
			if jsonMode || !Interactive() {
				return fmt.Errorf("purchase requires --yes in JSON or non-interactive mode")
			}
			fmt.Print(summary(r) + "\nProceed with airtime purchase? [y/N]: ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if strings.TrimSpace(strings.ToLower(line)) != "y" {
				return fmt.Errorf("purchase cancelled")
			}
		}
		r, e = s.Send(cmd.Context(), r)
		if jsonMode {
			if oe := output(r); oe != nil {
				return oe
			}
		} else {
			fmt.Printf("Operation: %s\nProvider ID: %s\nSubmission: %s\nDelivery: %s\n%s\n", r.ID, r.RequestID, r.Submission, r.Status, r.Message)
		}
		return e
	}}
	send.Flags().StringVar(&currency, "currency", "", "recipient currency (default from config)")
	send.Flags().BoolVar(&dry, "dry-run", false, "validate and preview only")
	send.Flags().BoolVar(&yes, "yes", false, "explicitly approve this purchase within safety limits")
	air.AddCommand(send)
	air.AddCommand(&cobra.Command{Use: "status ID", Short: "Check a local operation or provider ATQid", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		s, e := Open(env, true)
		if e != nil {
			return e
		}
		defer s.Store.DB.Close()
		r, e := s.Status(cmd.Context(), args[0])
		if e != nil {
			return e
		}
		if jsonMode {
			return output(r)
		}
		fmt.Printf("%s\nProvider ID: %s\nStatus: %s\nChecked: %s\nMessage: %s\n", summary(r), r.RequestID, r.Status, storage.LocalTime(r.Checked), r.Message)
		return nil
	}})
	root.AddCommand(air)
	var limit int
	var status string
	h := &cobra.Command{Use: "history", Short: "Locally recorded transactions", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if limit < 1 || limit > 1000 {
			return fmt.Errorf("limit must be 1–1000")
		}
		if status != "" && status != "pending" && status != "unknown" && status != "failed" && status != "delivered" {
			return fmt.Errorf("invalid status filter")
		}
		s, e := Open(env, false)
		if e != nil {
			return e
		}
		defer s.Store.DB.Close()
		rs, e := s.Store.List(cmd.Context(), s.Env, status, limit)
		if e != nil {
			return e
		}
		if jsonMode {
			return output(rs)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "DATE (LOCAL)\tRECIPIENT\tAMOUNT\tSTATUS\tENV\tOPERATION")
		for _, r := range rs {
			fmt.Fprintf(w, "%s\t%s\t%s %s\t%s\t%s\t%s\n", storage.LocalTime(r.Created), r.Recipient, r.Currency, money.Format(r.Amount), r.Status, r.Environment, r.ID)
		}
		return w.Flush()
	}}
	h.Flags().IntVar(&limit, "limit", 20, "maximum records")
	h.Flags().StringVar(&status, "status", "", "pending, delivered, failed or unknown")
	root.AddCommand(h)
	cfg := &cobra.Command{Use: "config", Short: "Manage settings and secure credentials"}
	cfg.AddCommand(&cobra.Command{Use: "show", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		c, e := config.Load()
		if e != nil {
			return e
		}
		return output(c)
	}})
	cfg.AddCommand(&cobra.Command{Use: "set KEY VALUE", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		c, e := config.Load()
		if e != nil {
			return e
		}
		switch args[0] {
		case "environment":
			c.Environment = args[1]
		case "default-currency":
			c.Currency = args[1]
		case "maximum":
			c.Maximum = args[1]
		case "daily-limit":
			c.Daily = args[1]
		case "username":
			target := env
			if target == "" {
				target = c.Environment
			}
			if target != "sandbox" && target != "production" {
				return fmt.Errorf("invalid environment")
			}
			c.Profiles[target] = config.Profile{Username: args[1]}
		default:
			return fmt.Errorf("unknown setting; use environment, default-currency, username, maximum or daily-limit")
		}
		return c.Save()
	}})
	cfg.AddCommand(&cobra.Command{Use: "test", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		s, e := Open(env, true)
		if e != nil {
			return e
		}
		defer s.Store.DB.Close()
		_, e = s.API.Balance(cmd.Context())
		if e == nil {
			if jsonMode {
				return output(map[string]bool{"connected": true})
			}
			fmt.Println("Credentials and balance connectivity verified; no airtime sent.")
		}
		return e
	}})
	cfg.AddCommand(&cobra.Command{Use: "init", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if jsonMode || !Interactive() {
			return fmt.Errorf("config init requires an interactive terminal; use config set and environment credentials for automation")
		}
		return tui.RunSetup(env)
	}})
	root.AddCommand(cfg)
	return root
}
func summary(r storage.Record) string {
	if r.Currency == "" {
		return fmt.Sprintf("Recipient: Not available\nAmount: Not available\nAccount: %s\nEnvironment: %s", r.Account, r.Environment)
	}
	return fmt.Sprintf("Recipient: %s\nAmount: %s %s\nNetwork: Not verified\nAccount: %s\nEnvironment: %s", r.Recipient, r.Currency, money.Format(r.Amount), r.Account, r.Environment)
}
func Execute() int {
	root := Root()
	e := root.ExecuteContext(context.Background())
	if e == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, "Error:", e)
	if errors.Is(e, service.ErrUnresolved) {
		return 5
	}
	if errors.Is(e, service.ErrRejected) {
		return 4
	}
	var apiErr *africastalking.Error
	if errors.As(e, &apiErr) {
		if apiErr.Code == 401 || apiErr.Code == 403 {
			return 3
		}
		return 4
	}
	return 1
}
