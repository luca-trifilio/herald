package tui

import (
	"database/sql"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luca-trifilio/herald/internal/claude"
	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/gmail"
	"github.com/luca-trifilio/herald/internal/models"
)

type DigestView struct {
	database    *sql.DB
	client      *claude.Client
	gmailClient *gmail.Client
	cfg         *models.Config
	items       []models.NewsItem
	cursor      int
	status      string
	fetching    bool
	showHelp    bool
	width       int
	height      int
}

func NewDigestView(database *sql.DB, client *claude.Client, gmailClient *gmail.Client, cfg *models.Config) *DigestView {
	return &DigestView{
		database:    database,
		client:      client,
		gmailClient: gmailClient,
		cfg:         cfg,
	}
}

func (v *DigestView) SetItems(items []models.NewsItem) {
	v.items = filterVisible(items)
	v.cursor = 0
}

func (v *DigestView) SetStatus(s string) { v.status = s }
func (v *DigestView) Items() []models.NewsItem { return v.items }
func (v *DigestView) SetSize(w, h int)         { v.width = w; v.height = h }

func filterVisible(items []models.NewsItem) []models.NewsItem {
	var out []models.NewsItem
	for _, it := range items {
		if !it.Discarded {
			out = append(out, it)
		}
	}
	return out
}

type fetchDoneMsg struct {
	digest *models.Digest
	err    error
}

type discardDoneMsg struct {
	itemID int64
	err    error
}

func (v *DigestView) fetchDigest(_ bool) tea.Cmd {
	return func() tea.Msg {
		digest, err := claude.BuildDigest(v.database, v.client, v.gmailClient, v.cfg)
		return fetchDoneMsg{digest: digest, err: err}
	}
}

func (v *DigestView) discardItem(item models.NewsItem) tea.Cmd {
	return func() tea.Msg {
		if err := db.DiscardItem(v.database, item.ID); err != nil {
			return discardDoneMsg{itemID: item.ID, err: err}
		}
		go func() { _ = claude.ExtractTopics(v.database, v.client, item) }()
		return discardDoneMsg{itemID: item.ID}
	}
}

func (v *DigestView) removeItem(item models.NewsItem) tea.Cmd {
	return func() tea.Msg {
		if err := db.DiscardItem(v.database, item.ID); err != nil {
			return discardDoneMsg{itemID: item.ID, err: err}
		}
		return discardDoneMsg{itemID: item.ID}
	}
}

func (v *DigestView) sortItems() {
	sort.Slice(v.items, func(i, j int) bool {
		return v.items[i].Priority < v.items[j].Priority
	})
}

func (v *DigestView) Init() tea.Cmd { return nil }

func (v *DigestView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width, v.height = msg.Width, msg.Height

	case fetchDoneMsg:
		v.fetching = false
		if msg.err != nil {
			v.status = "Error: " + msg.err.Error()
		} else if msg.digest == nil || len(msg.digest.Items) == 0 {
			v.status = "No new newsletters found."
		} else {
			v.SetItems(msg.digest.Items)
			v.status = fmt.Sprintf("%d items fetched", len(v.items))
		}

	case discardDoneMsg:
		if msg.err != nil {
			v.status = "Discard error: " + msg.err.Error()
		} else {
			var updated []models.NewsItem
			for _, it := range v.items {
				if it.ID != msg.itemID {
					updated = append(updated, it)
				}
			}
			v.items = updated
			if v.cursor >= len(v.items) && v.cursor > 0 {
				v.cursor--
			}
			v.status = "Item discarded."
		}

	case tea.KeyMsg:
		// Help toggle — highest priority
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

		if v.fetching {
			return v, nil
		}
		switch msg.String() {
		case "j", "down":
			if v.cursor < len(v.items)-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "enter":
			if len(v.items) > 0 {
				return v, pushDetailCmd(v.items[v.cursor])
			}
		case "d":
			if len(v.items) > 0 {
				return v, v.discardItem(v.items[v.cursor])
			}
		case "x":
			if len(v.items) > 0 {
				return v, v.removeItem(v.items[v.cursor])
			}
		case "o":
			if len(v.items) > 0 && v.items[v.cursor].URL != "" {
				exec.Command("open", v.items[v.cursor].URL).Start() //nolint
				v.status = "Opened in browser."
			} else {
				v.status = "No URL available."
			}
		case "+", "=":
			if len(v.items) > 0 {
				it := &v.items[v.cursor]
				if it.Priority > 1 {
					it.Priority--
					_ = db.UpdateItemPriority(v.database, it.ID, it.Priority)
					v.sortItems()
					// keep cursor on same item after sort
					for i, item := range v.items {
						if item.ID == it.ID {
							v.cursor = i
							break
						}
					}
					v.status = fmt.Sprintf("Priority set to %d.", v.items[v.cursor].Priority)
				}
			}
		case "-":
			if len(v.items) > 0 {
				it := &v.items[v.cursor]
				if it.Priority < 10 {
					it.Priority++
					_ = db.UpdateItemPriority(v.database, it.ID, it.Priority)
					v.sortItems()
					for i, item := range v.items {
						if item.ID == it.ID {
							v.cursor = i
							break
						}
					}
					v.status = fmt.Sprintf("Priority set to %d.", v.items[v.cursor].Priority)
				}
			}
		case "r":
			v.fetching = true
			v.status = "Fetching…"
			return v, v.fetchDigest(true)
		}
	}
	return v, nil
}

func (v *DigestView) View() string {
	if v.width == 0 {
		return ""
	}

	var b strings.Builder

	// ── Title bar ────────────────────────────────────────────────────────────
	appName := StyleTitleBarApp.Render("Herald")
	sub := StyleTitleBar.Copy().Foreground(colorMuted).Render(" — Tech Digest")
	titleBar := StyleTitleBar.Width(v.width).Render(appName + sub)
	b.WriteString(titleBar + "\n")

	// ── Body area (list + preview pane) ──────────────────────────────────────
	bodyH := v.height - 3 // subtract title bar + status line + keys line
	if bodyH < 1 {
		bodyH = 1
	}

	leftW := v.width * 40 / 100
	if leftW < 20 {
		leftW = 20
	}
	// divider is 1 char wide
	rightW := v.width - leftW - 1
	if rightW < 10 {
		rightW = 10
	}

	leftPane := v.renderList(leftW, bodyH)
	rightPane := v.renderPreview(rightW, bodyH)

	dividerStyle := lipgloss.NewStyle().Foreground(colorBorder)
	dividerLines := make([]string, bodyH)
	for i := range dividerLines {
		dividerLines[i] = dividerStyle.Render("│")
	}
	divider := strings.Join(dividerLines, "\n")

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, divider, rightPane)
	b.WriteString(body + "\n")

	// ── Bottom bar (status + keys) ────────────────────────────────────────────
	statusLine := v.status
	if statusLine == "" {
		statusLine = fmt.Sprintf("%d items", len(v.items))
	}
	b.WriteString(renderBottomBar(statusLine, digestHints, v.width))

	out := b.String()

	// ── Help overlay ──────────────────────────────────────────────────────────
	if v.showHelp {
		out = helpOverlay(out, "Digest — Keybindings", digestBindings, v.width, v.height)
	}

	return out
}

func (v *DigestView) renderList(w, h int) string {
	style := lipgloss.NewStyle().Width(w).Height(h)

	if v.fetching {
		content := "\n" + StyleMuted.Render("  Fetching digest from Gmail…")
		return style.Render(content)
	}
	if len(v.items) == 0 {
		content := "\n" + StyleMuted.Render("  No items. Press r to fetch.")
		return style.Render(content)
	}

	var rows []string
	start := 0
	if v.cursor >= h {
		start = v.cursor - h + 1
	}
	end := start + h
	if end > len(v.items) {
		end = len(v.items)
	}

	for i := start; i < end; i++ {
		it := v.items[i]
		if i == v.cursor {
			row := fmt.Sprintf("%2d %s — %s", it.Priority, it.Title, it.Source)
			// Width must subtract 2 for left border + padding
			rows = append(rows, StyleSelectedRow.Width(w-2).Render(row))
		} else {
			prioStyled := priorityStyle(it.Priority).Render(fmt.Sprintf("[%2d]", it.Priority))
			src := StyleMuted.Render(it.Source)
			// truncate title to fit
			maxTitle := w - 10 - len(it.Source)
			title := it.Title
			if len(title) > maxTitle && maxTitle > 3 {
				title = title[:maxTitle-1] + "…"
			}
			rows = append(rows, fmt.Sprintf("  %s %s — %s", prioStyled, title, src))
		}
	}

	content := strings.Join(rows, "\n")
	return style.Render(content)
}

func (v *DigestView) renderPreview(w, h int) string {
	style := lipgloss.NewStyle().
		Width(w).
		Height(h).
		PaddingLeft(2).
		PaddingRight(1)

	if len(v.items) == 0 || v.fetching {
		return style.Render("")
	}

	it := v.items[v.cursor]
	innerW := w - 3 // account for padding

	var b strings.Builder

	// Priority badge + title
	badge := priorityStyle(it.Priority).Bold(true).Render(fmt.Sprintf("[%d]", it.Priority))
	b.WriteString(badge + "\n\n")

	titleStyle := lipgloss.NewStyle().Foreground(colorFg).Bold(true).Width(innerW)
	b.WriteString(titleStyle.Render(it.Title) + "\n\n")

	// Source
	b.WriteString(StyleMuted.Render("Source: ") + StyleNormal.Render(it.Source) + "\n\n")

	// Summary
	b.WriteString(StyleKey.Render("Summary:") + "\n")
	summaryStyle := lipgloss.NewStyle().Foreground(colorFg).Width(innerW)
	b.WriteString(summaryStyle.Render(it.Summary) + "\n\n")

	// URL
	if it.URL != "" {
		b.WriteString(StyleKey.Render("URL:") + "\n")
		url := it.URL
		if len(url) > innerW && innerW > 4 {
			url = url[:innerW-1] + "…"
		}
		b.WriteString(StyleMuted.Render(url) + "\n\n")
	}

	// Hint
	hint := StyleMuted.Render("Enter detail • o browser • d discard • x remove • +/- priority")
	b.WriteString(hint)

	return style.Render(b.String())
}
