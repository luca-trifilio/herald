package tui

import (
	"database/sql"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luca-trifilio/herald/internal/db"
	"github.com/luca-trifilio/herald/internal/models"
)

type settingsSection int

const (
	sectionSources settingsSection = iota
	sectionPrefs
)

type settingsMode int

const (
	modeNav settingsMode = iota
	modeAddSourceName
	modeAddSourceEmail
)

type SettingsView struct {
	database *sql.DB
	sources  []models.NewsletterSource
	prefs    []models.Preference
	section  settingsSection
	cursor   int
	mode     settingsMode
	input    string
	newName  string
	status   string
	showHelp bool
	width    int
	height   int
}

func NewSettingsView(database *sql.DB) *SettingsView {
	v := &SettingsView{database: database}
	v.reload()
	return v
}

func (v *SettingsView) reload() {
	v.sources, _ = db.GetAllSources(v.database)
	v.prefs, _ = db.GetAllPreferences(v.database)
}

func (v *SettingsView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *SettingsView) Init() tea.Cmd { return nil }

func (v *SettingsView) currentListLen() int {
	if v.section == sectionSources {
		return len(v.sources)
	}
	return len(v.prefs)
}

func (v *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height

	case tea.KeyMsg:
		// Help toggle — always available
		if msg.String() == "?" && v.mode == modeNav {
			v.showHelp = !v.showHelp
			return v, nil
		}
		if v.showHelp {
			if msg.String() == "esc" {
				v.showHelp = false
			}
			return v, nil
		}

		if v.mode == modeAddSourceName {
			switch msg.String() {
			case "esc":
				v.mode = modeNav
				v.input = ""
				v.newName = ""
			case "enter":
				if v.input != "" {
					v.newName = v.input
					v.input = ""
					v.mode = modeAddSourceEmail
				}
			case "backspace":
				if len(v.input) > 0 {
					v.input = v.input[:len(v.input)-1]
				}
			default:
				if len(msg.String()) == 1 {
					v.input += msg.String()
				}
			}
			return v, nil
		}

		if v.mode == modeAddSourceEmail {
			switch msg.String() {
			case "esc":
				v.mode = modeNav
				v.input = ""
				v.newName = ""
			case "enter":
				if v.input != "" {
					if err := db.InsertSource(v.database, v.newName, v.input); err != nil {
						v.status = "Error: " + err.Error()
					} else {
						v.status = "Source added."
					}
					v.reload()
					v.mode = modeNav
					v.input = ""
					v.newName = ""
				}
			case "backspace":
				if len(v.input) > 0 {
					v.input = v.input[:len(v.input)-1]
				}
			default:
				if len(msg.String()) == 1 {
					v.input += msg.String()
				}
			}
			return v, nil
		}

		// modeNav
		switch msg.String() {
		case "esc", "b":
			return v, popViewCmd()
		case "tab":
			v.section = 1 - v.section
			v.cursor = 0
		case "j", "down":
			if v.cursor < v.currentListLen()-1 {
				v.cursor++
			}
		case "k", "up":
			if v.cursor > 0 {
				v.cursor--
			}
		case "a":
			if v.section == sectionSources {
				v.mode = modeAddSourceName
				v.input = ""
			}
		case "enter", " ":
			if v.section == sectionSources && len(v.sources) > 0 {
				s := v.sources[v.cursor]
				if err := db.ToggleSource(v.database, s.ID, !s.Active); err != nil {
					v.status = "Error: " + err.Error()
				} else {
					v.reload()
					v.status = "Source toggled."
				}
			}
		case "delete", "backspace":
			if v.section == sectionSources && len(v.sources) > 0 {
				s := v.sources[v.cursor]
				if err := db.DeleteSource(v.database, s.ID); err != nil {
					v.status = "Error: " + err.Error()
				} else {
					v.reload()
					if v.cursor >= len(v.sources) && v.cursor > 0 {
						v.cursor--
					}
					v.status = "Source removed."
				}
			} else if v.section == sectionPrefs && len(v.prefs) > 0 {
				p := v.prefs[v.cursor]
				if err := db.DeletePreference(v.database, p.ID); err != nil {
					v.status = "Error: " + err.Error()
				} else {
					v.reload()
					if v.cursor >= len(v.prefs) && v.cursor > 0 {
						v.cursor--
					}
					v.status = "Preference forgotten."
				}
			}
		}
	}
	return v, nil
}

func (v *SettingsView) View() string {
	var b strings.Builder

	// ── Title bar ─────────────────────────────────────────────────────────────
	titleBar := StyleTitleBar.Width(v.width).Render(
		StyleTitleBarApp.Render("Herald") +
			StyleTitleBar.Copy().Foreground(colorMuted).Render(" — Settings"),
	)
	b.WriteString(titleBar + "\n")
	b.WriteString(StyleMuted.PaddingLeft(2).Render("Tab to switch section  •  ? help  •  Esc/b back") + "\n\n")

	// ── Sources section ───────────────────────────────────────────────────────
	srcHeader := "Newsletter Sources"
	if v.section == sectionSources {
		srcHeader = StyleKey.Render("▶ ") + StyleSection.Render(srcHeader)
	} else {
		srcHeader = "  " + StyleSection.Render(srcHeader)
	}
	b.WriteString(srcHeader + "\n")

	if len(v.sources) == 0 {
		b.WriteString(StyleMuted.PaddingLeft(2).Render("(none)") + "\n")
	}
	for i, s := range v.sources {
		chk := StyleMuted.Render("[ ]")
		if s.Active {
			chk = StyleGreen.Render("[x]")
		}
		if v.section == sectionSources && i == v.cursor {
			row := fmt.Sprintf("%s %s — %s", "[ ]", s.Name, s.SenderEmail)
			if s.Active {
				row = fmt.Sprintf("%s %s — %s", "[x]", s.Name, s.SenderEmail)
			}
			b.WriteString(StyleSelectedRow.Width(v.width-2).Render(row) + "\n")
		} else {
			b.WriteString(fmt.Sprintf("  %s %s — %s\n", chk, s.Name, StyleMuted.Render(s.SenderEmail)))
		}
	}

	if v.mode == modeAddSourceName {
		b.WriteString(StyleMuted.PaddingLeft(2).Render("New source name: ") + StyleNormal.Render(v.input+"_") + "\n")
	} else if v.mode == modeAddSourceEmail {
		b.WriteString(StyleMuted.PaddingLeft(2).Render(fmt.Sprintf("Email for %q: ", v.newName)) + StyleNormal.Render(v.input+"_") + "\n")
	} else if v.section == sectionSources {
		b.WriteString(StyleHelp.PaddingLeft(2).Render("a add  •  Space toggle  •  Del remove") + "\n")
	}

	b.WriteString("\n")

	// ── Preferences section ───────────────────────────────────────────────────
	prefHeader := "Learned Preferences"
	if v.section == sectionPrefs {
		prefHeader = StyleKey.Render("▶ ") + StyleSection.Render(prefHeader)
	} else {
		prefHeader = "  " + StyleSection.Render(prefHeader)
	}
	b.WriteString(prefHeader + "\n")

	if len(v.prefs) == 0 {
		b.WriteString(StyleMuted.PaddingLeft(2).Render("(none learned yet)") + "\n")
	}
	for i, p := range v.prefs {
		if v.section == sectionPrefs && i == v.cursor {
			row := fmt.Sprintf("%s  weight %.1f  (seen %dx)", p.Topic, p.Weight, p.Occurrences)
			b.WriteString(StyleSelectedRow.Width(v.width-2).Render(row) + "\n")
		} else {
			line := fmt.Sprintf("  %s  weight %.1f  (seen %dx)", p.Topic, p.Weight, p.Occurrences)
			b.WriteString(StyleMuted.Render(line) + "\n")
		}
	}
	if v.section == sectionPrefs {
		b.WriteString(StyleHelp.PaddingLeft(2).Render("Del to forget a preference") + "\n")
	}

	// ── Bottom bar (status + keys) ────────────────────────────────────────────
	statusLine := v.status
	if statusLine == "" {
		statusLine = "Settings"
	}
	remaining := v.height - strings.Count(b.String(), "\n") - 2
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
	}
	b.WriteString(renderBottomBar(statusLine, settingsHints, v.width))

	out := b.String()

	if v.showHelp {
		out = helpOverlay(out, "Settings — Keybindings", settingsBindings, v.width, v.height)
	}

	return out
}
