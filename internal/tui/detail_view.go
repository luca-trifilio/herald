package tui

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/models"
)

type DetailView struct {
	item     models.NewsItem
	database *sql.DB
	status   string
	showHelp bool
	width    int
	height   int
}

func NewDetailView(item models.NewsItem, database *sql.DB) *DetailView {
	return &DetailView{item: item, database: database}
}

func (v *DetailView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *DetailView) Init() tea.Cmd {
	return func() tea.Msg {
		_ = db.MarkOpened(v.database, v.item.ID)
		return nil
	}
}

type popViewMsg struct{}
type discardAndPopMsg struct{ itemID int64 }

func (v *DetailView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	case tea.KeyMsg:
		if msg.String() == "?" {
			v.showHelp = !v.showHelp
			return v, nil
		}
		if v.showHelp {
			if msg.String() == "esc" {
				v.showHelp = false
			}
			return v, nil
		}
		switch msg.String() {
		case "esc", "b":
			return v, popViewCmd()
		case "o":
			if v.item.URL != "" {
				exec.Command("open", v.item.URL).Start() //nolint
				v.status = "Opened in browser."
			} else {
				v.status = "No URL available."
			}
		case "d":
			_ = db.DiscardItem(v.database, v.item.ID)
			return v, discardAndPopCmd(v.item.ID)
		}
	}
	return v, nil
}

func (v *DetailView) View() string {
	var b strings.Builder

	// ── Title bar ─────────────────────────────────────────────────────────────
	titleBar := StyleTitleBar.Width(v.width).Render(
		StyleTitleBarApp.Render("Herald") +
			StyleTitleBar.Copy().Foreground(colorMuted).Render(" — Detail"),
	)
	b.WriteString(titleBar + "\n\n")

	// ── Content ───────────────────────────────────────────────────────────────
	prioLabel := priorityStyle(v.item.Priority).Bold(true).Render(
		fmt.Sprintf("Priority %d", v.item.Priority),
	)
	b.WriteString("  " + prioLabel + "\n")
	b.WriteString("  " + StyleNormal.Bold(true).Width(v.width-4).Render(v.item.Title) + "\n")
	b.WriteString("  " + StyleMuted.Render("Source: "+v.item.Source) + "\n\n")

	// Summary wrapped to terminal width with padding
	b.WriteString(StyleNormal.Width(v.width-4).PaddingLeft(2).Render(v.item.Summary) + "\n\n")

	if v.item.URL != "" {
		b.WriteString("  " + StyleKey.Render("URL  ") + StyleMuted.Render(v.item.URL) + "\n")
	} else {
		b.WriteString("  " + StyleMuted.Render("URL: (none)") + "\n")
	}

	// ── Bottom bar (status + keys) ────────────────────────────────────────────
	statusLine := v.status
	if statusLine == "" {
		statusLine = "Detail view"
	}
	remaining := v.height - strings.Count(b.String(), "\n") - 2
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
	}
	b.WriteString(renderBottomBar(statusLine, detailHints, v.width))

	out := b.String()

	if v.showHelp {
		out = helpOverlay(out, "Detail — Keybindings", detailBindings, v.width, v.height)
	}

	return out
}

func StyleAccentRender(s string) string {
	return StyleKey.Render(s)
}
