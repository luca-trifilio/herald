package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// binding is a single key → description pair shown in the help overlay.
type binding struct {
	key  string
	desc string
}

// helpOverlay renders a centered modal with keybinding documentation layered
// on top of a background view string.
func helpOverlay(background string, title string, bindings []binding, termW, termH int) string {
	// Build the modal content
	var inner strings.Builder
	inner.WriteString(StyleHelpTitle.Render(title) + "\n\n")

	// Find max key length for alignment
	maxKey := 0
	for _, b := range bindings {
		if len(b.key) > maxKey {
			maxKey = len(b.key)
		}
	}

	for _, b := range bindings {
		padded := b.key + strings.Repeat(" ", maxKey-len(b.key))
		inner.WriteString(StyleHelpKey.Render(padded) + "  " + StyleHelpDesc.Render(b.desc) + "\n")
	}
	inner.WriteString("\n" + StyleHelpDesc.Render("? / Esc  dismiss"))

	modal := StyleHelpModal.Render(inner.String())

	// Calculate modal dimensions (lipgloss includes border in Size)
	modalLines := strings.Split(modal, "\n")
	modalH := len(modalLines)
	modalW := 0
	for _, l := range modalLines {
		if w := lipgloss.Width(l); w > modalW {
			modalW = w
		}
	}

	// Overlay the modal on the background by replacing lines in the center
	bgLines := strings.Split(background, "\n")

	// Ensure background has enough lines
	for len(bgLines) < termH {
		bgLines = append(bgLines, "")
	}

	topStart := (termH - modalH) / 2
	if topStart < 0 {
		topStart = 0
	}
	leftStart := (termW - modalW) / 2
	if leftStart < 0 {
		leftStart = 0
	}

	for i, mLine := range modalLines {
		row := topStart + i
		if row >= len(bgLines) {
			break
		}

		bg := bgLines[row]
		bgRunes := []rune(stripAnsi(bg))

		// Pad bg line to leftStart
		for len(bgRunes) < leftStart {
			bgRunes = append(bgRunes, ' ')
		}

		// Build new line: left portion + modal line + remainder if any
		leftPart := string(bgRunes[:leftStart])
		var rightPart string
		rightEnd := leftStart + modalW
		if rightEnd < len(bgRunes) {
			rightPart = string(bgRunes[rightEnd:])
		}
		bgLines[row] = leftPart + mLine + rightPart
	}

	return strings.Join(bgLines, "\n")
}

// stripAnsi removes ANSI escape sequences to get the plain rune count.
// Minimal implementation — handles CSI sequences (ESC [ ... m) only.
func stripAnsi(s string) string {
	var out strings.Builder
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// skip until 'm' or other terminator
			i += 2
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				i++
			}
			i++ // skip terminator
			continue
		}
		out.WriteRune(runes[i])
		i++
	}
	return out.String()
}

func isAnsiTerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// ── Two-line bottom bar ───────────────────────────────────────────────────────

// keysHint is a single key→desc pair for the always-visible keys bar.
type keysHint struct {
	key  string
	desc string
}

// renderBottomBar returns the two-line bottom bar string (status line + keys
// line), each padded/truncated to termW.
//
//	line 1: statusLine (current status message or idle text)
//	line 2: rendered keybinding hints separated by " · "
func renderBottomBar(statusLine string, hints []keysHint, termW int) string {
	sep := StyleKeySep.Render(" · ")

	var parts []string
	for _, h := range hints {
		chunk := StyleKeyLabel.Render(h.key) + " " + StyleKeyDesc.Render(h.desc)
		parts = append(parts, chunk)
	}

	keysLine := strings.Join(parts, sep)

	s1 := StyleStatusBar.Width(termW).Render(statusLine)
	s2 := StyleKeysBar.Width(termW).Render(keysLine)
	return s1 + "\n" + s2
}

// ── Per-view binding tables ───────────────────────────────────────────────────

var digestHints = []keysHint{
	{"j/k", "nav"},
	{"enter", "open"},
	{"o", "browser"},
	{"d", "discard"},
	{"x", "remove"},
	{"+/-", "priority"},
	{"r", "refresh"},
	{"s", "settings"},
	{"?", "help"},
	{"q", "quit"},
}

var detailHints = []keysHint{
	{"o", "browser"},
	{"d", "discard"},
	{"b", "back"},
	{"?", "help"},
}

var settingsHints = []keysHint{
	{"j/k", "nav"},
	{"tab", "switch"},
	{"a", "add"},
	{"del", "remove"},
	{"space", "toggle"},
	{"b", "back"},
	{"?", "help"},
}

var digestBindings = []binding{
	{"j / k", "navigate down / up"},
	{"Enter", "open detail view"},
	{"o", "open URL in browser"},
	{"d", "discard item (learn topics)"},
	{"x", "remove item (no learning)"},
	{"+  /  -", "increase / decrease priority"},
	{"r", "refresh digest"},
	{"s", "open settings"},
	{"?", "toggle help"},
	{"q", "quit"},
}

var detailBindings = []binding{
	{"o", "open URL in browser"},
	{"d", "discard & go back"},
	{"b / Esc", "back to list"},
	{"?", "toggle help"},
}

var settingsBindings = []binding{
	{"j / k", "navigate down / up"},
	{"Tab", "switch section"},
	{"Space / Enter", "toggle source"},
	{"a", "add new source"},
	{"Del / Backspace", "remove item"},
	{"b / Esc", "back to digest"},
	{"?", "toggle help"},
}
