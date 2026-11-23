package tui

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nasoma/atcli/internal/africastalking"
	"github.com/nasoma/atcli/internal/config"
	"github.com/nasoma/atcli/internal/money"
	"github.com/nasoma/atcli/internal/service"
	"github.com/nasoma/atcli/internal/storage"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var menu = []string{"Send Airtime", "Check Balance", "Transaction History", "Transaction Status", "Settings", "Exit"}

type result struct {
	text    string
	err     error
	record  *storage.Record
	records []storage.Record
	setup   bool
}
type model struct {
	cfg                            config.Config
	env                            string
	store                          *storage.Store
	screen                         string
	selected, focus, width, height int
	fields                         []textinput.Model
	spin                           spinner.Model
	busy                           bool
	notice                         string
	noticeError                    bool
	preview                        storage.Record
	rows                           []storage.Record
	table                          table.Model
	first                          bool
}

func Run(env string) error      { return run(env, false) }
func RunSetup(env string) error { return run(env, true) }
func run(env string, setup bool) error {
	c, e := config.Load()
	if e != nil {
		return e
	}
	if env == "" {
		env = os.Getenv("AT_ENVIRONMENT")
	}
	if env == "" {
		env = c.Environment
	}
	if env != "sandbox" && env != "production" {
		return fmt.Errorf("invalid environment")
	}
	d, e := config.Dir()
	if e != nil {
		return e
	}
	st, e := storage.Open(filepath.Join(d, "history.sqlite"))
	if e != nil {
		return e
	}
	defer st.DB.Close()
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(teal)
	m := model{cfg: c, env: env, store: st, screen: "dashboard", width: 80, height: 24, spin: sp, first: !config.Exists() || c.Profiles[env].Username == ""}
	if setup || m.first {
		m.settings()
	}
	_, e = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return e
}
func (m model) Init() tea.Cmd { return m.spin.Tick }
func (m *model) form(screen string, labels, values []string) {
	m.screen = screen
	m.fields = nil
	m.focus = 0
	m.notice = ""
	m.noticeError = false
	for i, label := range labels {
		f := textinput.New()
		f.Placeholder = ""
		switch label {
		case "Recipient":
			f.Placeholder = "+254712345678"
		case "Amount":
			f.Placeholder = "100.00"
		case "API key":
			f.Placeholder = "Enter key, or blank to keep"
		case "Operation or provider ID":
			f.Placeholder = "ATQid_…"
		}
		f.Prompt = label + ": "
		f.PromptStyle = headingStyle
		f.Cursor.Style = lipgloss.NewStyle().Foreground(teal)
		f.CharLimit = 200
		f.Width = 32
		if i < len(values) {
			f.SetValue(values[i])
		}
		if label == "API key" {
			f.EchoMode = textinput.EchoPassword
			f.EchoCharacter = '•'
		}
		if i == 0 {
			f.Focus()
		}
		m.fields = append(m.fields, f)
	}
}
func (m *model) settings() {
	m.form("settings", []string{"Environment", "Username", "API key", "Currency", "Maximum", "Daily limit (0 disables)"}, []string{m.env, m.username(), "", m.cfg.Currency, m.cfg.Maximum, m.cfg.Daily})
}
func (m model) svc() (*service.Service, error) {
	u, k, e := m.cfg.Credentials(m.env)
	if e != nil {
		return nil, e
	}
	a, e := africastalking.New(m.env, u, k)
	return &service.Service{Config: m.cfg, Env: m.env, User: u, API: a, Store: m.store}, e
}
func (m model) action(kind string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		s, e := m.svc()
		if e != nil {
			return result{err: e}
		}
		switch kind {
		case "balance":
			b, e := s.API.Balance(ctx)
			return result{text: "Account balance: " + b.Currency + " " + b.Amount, err: e}
		case "send":
			r, e := s.Send(ctx, m.preview)
			if e == nil && r.Status == "pending" && r.RequestID != "" {
				if checked, checkErr := s.Status(ctx, r.ID); checkErr == nil {
					r = checked
				}
			}
			return result{record: &r, err: e}
		case "status":
			r, e := s.Status(ctx, m.fields[0].Value())
			return result{record: &r, err: e}
		}
		return result{}
	}
}
func (m model) saveSettings() tea.Cmd {
	values := make([]string, len(m.fields))
	for i, f := range m.fields {
		values[i] = f.Value()
	}
	return func() tea.Msg {
		c := m.cfg
		env := values[0]
		c.Environment = env
		c.Currency = values[3]
		c.Maximum = values[4]
		c.Daily = values[5]
		if e := c.Validate(); e != nil {
			return result{err: e}
		}
		if env == "sandbox" && values[1] != "sandbox" {
			return result{err: fmt.Errorf("sandbox username must be sandbox")}
		}
		c.Profiles = map[string]config.Profile{}
		for k, v := range m.cfg.Profiles {
			c.Profiles[k] = v
		}
		c.Profiles[env] = config.Profile{Username: values[1]}
		if values[2] != "" {
			if e := config.StoreKey(env, values[1], values[2]); e != nil {
				return result{err: e}
			}
		}
		u, k, e := c.Credentials(env)
		if e != nil {
			return result{err: e}
		}
		a, e := africastalking.New(env, u, k)
		if e != nil {
			return result{err: e}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if _, e = a.Balance(ctx); e != nil {
			return result{err: e}
		}
		if e = c.Save(); e != nil {
			return result{err: e}
		}
		return result{setup: true, text: "Settings saved; credentials verified without sending airtime."}
	}
}
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
		for i := range m.fields {
			m.fields[i].Width = max(8, min(32, m.width-30))
		}
		m.table.SetHeight(max(1, min(12, m.height-10)))
		m.table.SetWidth(max(1, m.width-8))
		return m, nil
	case result:
		m.busy = false
		m.noticeError = v.err != nil
		if v.err != nil {
			m.notice = v.err.Error()
		} else {
			m.notice = v.text
		}
		if v.setup {
			c, e := config.Load()
			if e == nil {
				m.cfg = c
				m.env = c.Environment
			}
			m.first = false
			m.screen = "dashboard"
			m.fields = nil
		}
		if v.record != nil {
			r := *v.record
			m.screen = "result"
			m.notice = recordText(r) + "\n" + m.notice
		}
		if v.records != nil {
			m.rows = v.records
			rows := []table.Row{}
			for _, r := range m.rows {
				date := r.Created
				if t, e := time.Parse(time.RFC3339Nano, r.Created); e == nil {
					date = t.Local().Format("01-02 15:04")
				}
				rows = append(rows, table.Row{date, r.Recipient, r.Currency + " " + money.Format(r.Amount), transactionStatus(r)})
			}
			m.table = table.New(table.WithColumns([]table.Column{{Title: "Local date", Width: 11}, {Title: "Recipient", Width: 14}, {Title: "Amount", Width: 12}, {Title: "Status", Width: 9}}), table.WithRows(rows), table.WithHeight(max(1, min(12, m.height-10))), table.WithFocused(true), table.WithWidth(max(1, m.width-8)))
			styles := table.DefaultStyles()
			styles.Header = styles.Header.Foreground(teal).Bold(true)
			styles.Selected = styles.Selected.Foreground(teal).Background(lipgloss.AdaptiveColor{Light: "254", Dark: "236"}).Bold(true)
			m.table.SetStyles(styles)
			m.screen = "history"
		}
		return m, nil
	case tea.KeyMsg:
		if v.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.width > 0 && (m.width < 45 || m.height < 15) {
			if v.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.busy {
			return m, nil
		}
		if v.String() == "esc" {
			m.screen = "dashboard"
			m.notice = ""
			m.noticeError = false
			m.fields = nil
			return m, nil
		}
		switch m.screen {
		case "dashboard":
			switch v.String() {
			case "q":
				return m, tea.Quit
			case "up", "k":
				m.selected = (m.selected + len(menu) - 1) % len(menu)
			case "down", "j":
				m.selected = (m.selected + 1) % len(menu)
			case "enter":
				m.notice = ""
				m.noticeError = false
				switch m.selected {
				case 0:
					m.form("send", []string{"Recipient", "Amount", "Currency"}, []string{"", "", m.cfg.Currency})
					return m, textinput.Blink
				case 1:
					m.busy = true
					return m, m.action("balance")
				case 2:
					m.busy = true
					return m, func() tea.Msg {
						rs, e := m.store.List(context.Background(), m.env, "", 100)
						return result{records: rs, err: e}
					}
				case 3:
					m.form("status", []string{"Operation or provider ID"}, nil)
					return m, textinput.Blink
				case 4:
					m.settings()
					return m, textinput.Blink
				case 5:
					return m, tea.Quit
				}
			}
		case "send", "settings", "status":
			if v.String() == "tab" || v.String() == "down" || v.String() == "up" {
				m.fields[m.focus].Blur()
				step := 1
				if v.String() == "up" {
					step = -1
				}
				m.focus = (m.focus + step + len(m.fields)) % len(m.fields)
				m.fields[m.focus].Focus()
				return m, textinput.Blink
			}
			if v.String() == "enter" {
				switch m.screen {
				case "send":
					s := service.Service{Config: m.cfg, Env: m.env, User: m.username()}
					r, e := s.Preview(m.fields[0].Value(), m.fields[1].Value(), m.fields[2].Value())
					if e != nil {
						m.notice = e.Error()
						m.noticeError = true
					} else {
						m.preview = r
						m.screen = "confirm"
					}
				case "settings":
					m.busy = true
					return m, m.saveSettings()
				case "status":
					m.busy = true
					return m, m.action("status")
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
			return m, cmd
		case "confirm":
			if v.String() == "enter" {
				m.busy = true
				return m, m.action("send")
			}
			if v.String() == "n" {
				m.screen = "send"
			}
		case "history":
			if v.String() == "enter" && len(m.rows) > 0 {
				m.notice = recordText(m.rows[m.table.Cursor()])
				m.noticeError = false
				m.screen = "result"
				return m, nil
			}
			if v.String() == "q" {
				m.screen = "dashboard"
				return m, nil
			}
			var cmd tea.Cmd
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		case "result":
			if v.String() == "q" || v.String() == "enter" {
				m.screen = "dashboard"
				m.notice = ""
				m.noticeError = false
			}
		}
	}
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}

// transactionStatus separates a successful submission from confirmed delivery.
func transactionStatus(r storage.Record) string {
	switch r.Status {
	case "delivered":
		return "Success"
	case "pending":
		if r.Submission == "accepted" && strings.EqualFold(r.Message, "Sent") {
			return "Sent"
		}
		return "Pending"
	case "failed":
		return "Failed"
	default:
		return "Unknown"
	}
}

func recordText(r storage.Record) string {
	status := transactionStatus(r)
	if status == "Sent" {
		status += " (accepted; delivery not yet confirmed)"
	}
	if r.Currency == "" {
		return fmt.Sprintf("Provider ID: %s\nRecipient/amount: Not available\nStatus: %s\nLast checked: %s\nMessage: %s", r.RequestID, status, storage.LocalTime(r.Checked), r.Message)
	}
	return fmt.Sprintf("Operation: %s\nProvider ID: %s\nRecipient: %s\nAmount: %s %s\nSubmission: %s\nStatus: %s\nLast checked: %s\nMessage: %s", r.ID, r.RequestID, r.Recipient, r.Currency, money.Format(r.Amount), r.Submission, status, storage.LocalTime(r.Checked), r.Message)
}
func (m model) View() string {
	if m.width > 0 && (m.width < 45 || m.height < 15) {
		return "ATCLI — enlarge terminal to at least 45×15.\n80×24 recommended. q / Ctrl+C quits.\n"
	}
	title := headingStyle.Render("ATCLI — Africa's Talking")
	if m.env == "sandbox" {
		title += "  •  " + headingStyle.Render("SANDBOX")
	}
	body := ""
	switch m.screen {
	case "dashboard":
		for i, item := range menu {
			prefix := "  "
			if i == m.selected {
				prefix = "› "
			}
			line := prefix + item
			if i == m.selected {
				line = headingStyle.Render(line)
			}
			body += line + "\n"
		}
		body += "\n↑↓ Navigate  Enter Select  q Quit"
	case "send", "settings", "status":
		if m.first {
			body = "Welcome to ATCLI\nConnect your Africa’s Talking production application.\nPurchases send real airtime and charge your account.\nAPI keys use the OS credential store.\n\n"
		}
		for _, f := range m.fields {
			body += f.View() + "\n"
		}
		body += "\nTab/↑↓ Change field  Enter Continue / Save & Test  Esc Back"
		if m.screen == "settings" {
			body += "\nChanging environment also requires its matching username and key.\nBlank API key keeps the stored key; AT_* variables take precedence."
		}
	case "confirm":
		body = warningStyle.Render("CONFIRM PURCHASE") + fmt.Sprintf("\n\nRecipient: %s\nAmount: %s %s\nNetwork: Not verified\n\nPress Enter to purchase, n to edit, Esc to cancel.\nSent means accepted, not delivered.", m.preview.Recipient, m.preview.Currency, money.Format(m.preview.Amount))
	case "history":
		body += "Locally recorded purchases\n" + m.table.View() + "\n↑↓ Select  Enter Details  Esc Back"
	case "result":
		body += "Transaction details\n"
	}
	if m.busy {
		body += "\n" + m.spin.View() + " Working… purchase controls locked."
	}
	if m.notice != "" {
		notice := m.notice
		if m.noticeError {
			notice = errorStyle.Render(notice)
		}
		body += "\n\n" + notice
	}
	if m.screen != "dashboard" {
		body += "\n\nEsc Back  Ctrl+C Quit"
	}
	w := max(12, min(86, m.width-4))
	return lipgloss.NewStyle().Width(w).Border(lipgloss.RoundedBorder()).BorderForeground(teal).Padding(0, 1).Render(title+"\n\n"+body) + "\n"
}

func (m model) username() string {
	if v := os.Getenv("AT_USERNAME"); v != "" {
		return v
	}
	return m.cfg.Profiles[m.env].Username
}
