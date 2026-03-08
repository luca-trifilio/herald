package tui

import (
	"database/sql"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luca-trifilio/herald/internal/claude"
	"github.com/luca-trifilio/herald/internal/gmail"
	"github.com/luca-trifilio/herald/internal/models"
)

type pushDetailMsgType struct{ item models.NewsItem }
type pushSettingsMsgType struct{}
type popViewMsgType struct{}
type discardAndPopMsgType struct{ itemID int64 }

func pushDetailCmd(item models.NewsItem) tea.Cmd {
	return func() tea.Msg { return pushDetailMsgType{item: item} }
}

func pushSettingsCmd() tea.Cmd {
	return func() tea.Msg { return pushSettingsMsgType{} }
}

func popViewCmd() tea.Cmd {
	return func() tea.Msg { return popViewMsgType{} }
}

func discardAndPopCmd(itemID int64) tea.Cmd {
	return func() tea.Msg { return discardAndPopMsgType{itemID: itemID} }
}

type App struct {
	database   *sql.DB
	client     *claude.Client
	cfg        *models.Config
	digestView *DigestView
	stack      []tea.Model
	noAPIKey   bool
	noGmail    bool
	width      int
	height     int
}

func NewApp(database *sql.DB, client *claude.Client, gmailClient *gmail.Client, cfg *models.Config) *App {
	return &App{
		database:   database,
		client:     client,
		cfg:        cfg,
		digestView: NewDigestView(database, client, gmailClient, cfg),
		noAPIKey:   cfg.AnthropicAPIKey == "",
		noGmail:    gmailClient == nil,
	}
}

func (a *App) current() tea.Model {
	if len(a.stack) > 0 {
		return a.stack[len(a.stack)-1]
	}
	return a.digestView
}

func (a *App) Init() tea.Cmd {
	if a.noAPIKey || a.noGmail {
		return nil
	}
	return a.digestView.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.digestView.SetSize(msg.Width, msg.Height)
		for _, v := range a.stack {
			type sizer interface{ SetSize(int, int) }
			if s, ok := v.(sizer); ok {
				s.SetSize(msg.Width, msg.Height)
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if len(a.stack) == 0 {
				return a, tea.Quit
			}
		case "s":
			if len(a.stack) == 0 && !a.noAPIKey {
				sv := NewSettingsView(a.database)
				sv.SetSize(a.width, a.height)
				a.stack = append(a.stack, sv)
				return a, sv.Init()
			}
		}

	case pushDetailMsgType:
		dv := NewDetailView(msg.item, a.database)
		dv.SetSize(a.width, a.height)
		a.stack = append(a.stack, dv)
		return a, dv.Init()

	case pushSettingsMsgType:
		sv := NewSettingsView(a.database)
		sv.SetSize(a.width, a.height)
		a.stack = append(a.stack, sv)
		return a, sv.Init()

	case popViewMsgType:
		if len(a.stack) > 0 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, nil

	case discardAndPopMsgType:
		if len(a.stack) > 0 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		var updated []models.NewsItem
		for _, it := range a.digestView.items {
			if it.ID != msg.itemID {
				updated = append(updated, it)
			}
		}
		a.digestView.items = updated
		if a.digestView.cursor >= len(a.digestView.items) && a.digestView.cursor > 0 {
			a.digestView.cursor--
		}
		a.digestView.status = "Item discarded."
		return a, nil
	}

	cur := a.current()
	_, cmd := cur.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	if a.noAPIKey {
		return fmt.Sprintf(`
  %s

  No Anthropic API key found.

  Set it with:
    export ANTHROPIC_API_KEY=your-key-here

  Or add "anthropic_api_key" to ~/.config/newsdigest/config.json
  Then re-run herald.

  %s
`, StyleTitle.Render("Herald — Tech Digest"), StyleHelp.Render("q quit"))
	}

	if a.noGmail {
		return fmt.Sprintf(`
  %s

  Gmail not connected. Run once:

    ./herald --setup-gmail

  %s
`, StyleTitle.Render("Herald — Tech Digest"), StyleHelp.Render("q quit"))
	}

	return a.current().View()
}

func SetTriggerFetch(a *App) tea.Cmd {
	return a.digestView.fetchDigest(false)
}

func (a *App) GetDigestView() *DigestView {
	return a.digestView
}
