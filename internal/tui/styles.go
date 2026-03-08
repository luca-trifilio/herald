package tui

import "github.com/charmbracelet/lipgloss"

// Palette — Tokyo Night–inspired dark theme
var (
	colorBg       = lipgloss.Color("#1a1b26")
	colorBgAlt    = lipgloss.Color("#24283b")
	colorFg       = lipgloss.Color("#c0caf5")
	colorMuted    = lipgloss.Color("#565f89")
	colorAccent   = lipgloss.Color("#7dcfff")
	colorBlue     = lipgloss.Color("#7aa2f7")
	colorGreen    = lipgloss.Color("#9ece6a")
	colorTeal     = lipgloss.Color("#2ac3de")
	colorAmber    = lipgloss.Color("#ff9e64")
	colorYellow   = lipgloss.Color("#e0af68")
	colorRed      = lipgloss.Color("#f7768e")
	colorPurple   = lipgloss.Color("#bb9af7")
	colorBorder   = lipgloss.Color("#3b4261")
	colorSelected = lipgloss.Color("#2d3f76")
	colorBarBg    = lipgloss.Color("#16161e")
)

// ── Title bar ────────────────────────────────────────────────────────────────

var StyleTitleBar = lipgloss.NewStyle().
	Background(colorBgAlt).
	Foreground(colorAccent).
	Bold(true).
	PaddingLeft(2).
	PaddingRight(2)

var StyleTitleBarApp = lipgloss.NewStyle().
	Background(colorBgAlt).
	Foreground(colorBlue).
	Bold(true)

// Backwards-compat alias used in app.go / detail_view.go
var StyleTitle = lipgloss.NewStyle().
	Bold(true).
	Foreground(colorAccent).
	MarginBottom(1)

var StyleSubtitle = lipgloss.NewStyle().
	Foreground(colorMuted).
	Italic(true)

// ── List items ────────────────────────────────────────────────────────────────

// Selected row: left accent border + highlight background
var StyleSelectedRow = lipgloss.NewStyle().
	BorderLeft(true).
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(colorAccent).
	Background(colorSelected).
	Foreground(colorFg).
	Bold(true).
	PaddingLeft(1)

// Alias kept for legacy callers inside the package
var StyleSelected = StyleSelectedRow

var StyleNormal = lipgloss.NewStyle().
	Foreground(colorFg)

var StyleMuted = lipgloss.NewStyle().
	Foreground(colorMuted)

// ── Accent / semantic colours ─────────────────────────────────────────────────

var StyleAmber = lipgloss.NewStyle().
	Foreground(colorAmber).
	Bold(true)

var StyleGreen = lipgloss.NewStyle().
	Foreground(colorGreen)

var StyleRed = lipgloss.NewStyle().
	Foreground(colorRed)

var StyleKey = lipgloss.NewStyle().
	Foreground(colorAccent).
	Bold(true)

var StyleSection = lipgloss.NewStyle().
	Foreground(colorBlue).
	Bold(true)

// ── Status / keys bar ─────────────────────────────────────────────────────────

var StyleStatusBar = lipgloss.NewStyle().
	Background(colorBarBg).
	Foreground(colorMuted).
	PaddingLeft(2).
	PaddingRight(2)

// StyleKeysBar is the background style for the keys-hint line.
var StyleKeysBar = lipgloss.NewStyle().
	Background(colorBarBg).
	Foreground(colorMuted).
	PaddingLeft(2).
	PaddingRight(2)

// StyleKeyLabel — teal/cyan, bold: the key name (e.g. "j/k").
var StyleKeyLabel = lipgloss.NewStyle().
	Background(colorBarBg).
	Foreground(colorTeal).
	Bold(true)

// StyleKeyDesc — light foreground: the description text (e.g. "nav").
var StyleKeyDesc = lipgloss.NewStyle().
	Background(colorBarBg).
	Foreground(colorFg)

// StyleKeySep — muted separator "·".
var StyleKeySep = lipgloss.NewStyle().
	Background(colorBarBg).
	Foreground(colorMuted)

// ── Borders / containers ──────────────────────────────────────────────────────

var StyleBorder = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorBorder).
	Padding(0, 1)

// ── Help overlay modal ────────────────────────────────────────────────────────

var StyleHelpModal = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorAccent).
	Background(colorBgAlt).
	Padding(1, 3)

var StyleHelpTitle = lipgloss.NewStyle().
	Background(colorBgAlt).
	Foreground(colorAccent).
	Bold(true).
	MarginBottom(1)

var StyleHelpKey = lipgloss.NewStyle().
	Background(colorBgAlt).
	Foreground(colorTeal).
	Bold(true)

var StyleHelpDesc = lipgloss.NewStyle().
	Background(colorBgAlt).
	Foreground(colorMuted)

// legacy
var StyleHelp = lipgloss.NewStyle().
	Foreground(colorMuted)

// ── Priority colour gradient ──────────────────────────────────────────────────

func priorityStyle(p int) lipgloss.Style {
	switch {
	case p <= 3:
		// green/teal — high priority
		return lipgloss.NewStyle().Foreground(colorTeal).Bold(true)
	case p <= 6:
		// yellow — medium
		return lipgloss.NewStyle().Foreground(colorYellow)
	default:
		// muted — low priority
		return StyleMuted
	}
}
