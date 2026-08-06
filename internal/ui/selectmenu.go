package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/emreerinc/buum/pkg/buum"
)

// ErrSelectCanceled is returned by SelectManagers when the user cancels
// (q, esc, or ctrl-c) instead of confirming a selection.
var ErrSelectCanceled = errors.New("selection canceled")

// selectModel is the pure state of the interactive checkbox menu. checked is
// indexed to items (original order) and persists across filter changes; cursor
// indexes into the currently visible slice.
type selectModel struct {
	items     []string
	checked   []bool
	cursor    int
	filter    string
	filtering bool
}

func newSelectModel(items []string) *selectModel {
	checked := make([]bool, len(items))
	for i := range checked {
		checked[i] = true
	}
	return &selectModel{items: items, checked: checked}
}

// visible returns original indices whose name contains filter (case-insensitive).
func (m *selectModel) visible() []int {
	f := strings.ToLower(m.filter)
	var out []int
	for i, name := range m.items {
		if f == "" || strings.Contains(strings.ToLower(name), f) {
			out = append(out, i)
		}
	}
	return out
}

func (m *selectModel) clampCursor() {
	n := len(m.visible())
	switch {
	case n == 0:
		m.cursor = 0
	case m.cursor < 0:
		m.cursor = 0
	case m.cursor >= n:
		m.cursor = n - 1
	}
}

func (m *selectModel) moveUp() {
	if n := len(m.visible()); n > 0 {
		m.cursor = (m.cursor - 1 + n) % n
	}
}

func (m *selectModel) moveDown() {
	if n := len(m.visible()); n > 0 {
		m.cursor = (m.cursor + 1) % n
	}
}

func (m *selectModel) toggle() {
	vis := m.visible()
	if len(vis) == 0 {
		return
	}
	orig := vis[m.cursor]
	m.checked[orig] = !m.checked[orig]
}

// toggleAll checks all visible items, or clears them if all are already checked.
func (m *selectModel) toggleAll() {
	vis := m.visible()
	if len(vis) == 0 {
		return
	}
	allChecked := true
	for _, i := range vis {
		if !m.checked[i] {
			allChecked = false
			break
		}
	}
	for _, i := range vis {
		m.checked[i] = !allChecked
	}
}

func (m *selectModel) setFilter(s string) {
	m.filter = s
	m.clampCursor()
}

func (m *selectModel) selected() []string {
	// Non-nil even when empty: a deselect-all is an explicit "run nothing",
	// which BuildPlan distinguishes from a nil (unfiltered) selection.
	out := []string{}
	for i, name := range m.items {
		if m.checked[i] {
			out = append(out, name)
		}
	}
	return out
}

func (m *selectModel) counts() (sel, total int) {
	for _, c := range m.checked {
		if c {
			sel++
		}
	}
	return sel, len(m.items)
}

// render returns the full menu frame. Lines use CRLF for raw-terminal mode.
func (m *selectModel) render() string {
	var b strings.Builder
	vis := m.visible()

	if m.filtering {
		fmt.Fprintf(&b, "/ %s    %d matches\r\n", m.filter, len(vis))
	} else {
		sel, total := m.counts()
		fmt.Fprintf(&b, "? Select managers to update    %d of %d selected\r\n", sel, total)
	}
	b.WriteString("  ↑↓ move · space toggle · a all · / filter · enter run · q cancel\r\n")

	if len(vis) == 0 {
		b.WriteString("  (no matches)\r\n")
		return b.String()
	}
	for i, orig := range vis {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		box := "[ ]"
		if m.checked[orig] {
			box = "[x]"
		}
		fmt.Fprintf(&b, "%s%s %s\r\n", cursor, box, m.items[orig])
	}
	return b.String()
}

type key int

const (
	keyNone key = iota
	keyUp
	keyDown
	keyEnter
	keyEsc
	keyBackspace
	keyCtrlC
	keySpace
	keyPrintable
)

// keyEvent is a decoded keypress. r is set only when kind == keyPrintable.
type keyEvent struct {
	kind key
	r    rune
}

// decodeKey maps a raw byte sequence to a keyEvent. It is purely mechanical:
// context-sensitive meaning (e.g. 'a' = toggle-all only outside filter mode) is
// resolved in apply, not here.
func decodeKey(b []byte) keyEvent {
	if len(b) == 0 {
		return keyEvent{kind: keyNone}
	}
	if b[0] == 0x1b {
		if len(b) >= 3 && b[1] == '[' {
			switch b[2] {
			case 'A':
				return keyEvent{kind: keyUp}
			case 'B':
				return keyEvent{kind: keyDown}
			}
			return keyEvent{kind: keyNone}
		}
		return keyEvent{kind: keyEsc}
	}
	switch b[0] {
	case 0x03:
		return keyEvent{kind: keyCtrlC}
	case 0x0d, 0x0a:
		return keyEvent{kind: keyEnter}
	case 0x20:
		return keyEvent{kind: keySpace}
	case 0x7f, 0x08:
		return keyEvent{kind: keyBackspace}
	}
	if b[0] >= 0x21 && b[0] < 0x7f {
		return keyEvent{kind: keyPrintable, r: rune(b[0])}
	}
	return keyEvent{kind: keyNone}
}

type action int

const (
	actNone action = iota
	actConfirm
	actCancel
)

// apply mutates the model for a keyEvent and returns a high-level action.
// In filter mode, printable keys (including 'a', 'q', '/') are literal filter
// input; enter/esc exit filter mode keeping the filter.
func apply(m *selectModel, e keyEvent) action {
	if m.filtering {
		switch e.kind {
		case keyEnter, keyEsc:
			m.filtering = false
		case keyBackspace:
			if r := []rune(m.filter); len(r) > 0 {
				m.setFilter(string(r[:len(r)-1]))
			}
		case keyUp:
			m.moveUp()
		case keyDown:
			m.moveDown()
		case keyCtrlC:
			return actCancel
		case keyPrintable:
			m.setFilter(m.filter + string(e.r))
		}
		return actNone
	}
	switch e.kind {
	case keyUp:
		m.moveUp()
	case keyDown:
		m.moveDown()
	case keySpace:
		m.toggle()
	case keyEnter:
		return actConfirm
	case keyEsc, keyCtrlC:
		return actCancel
	case keyPrintable:
		switch e.r {
		case 'a':
			m.toggleAll()
		case 'q':
			return actCancel
		case '/':
			m.filtering = true
		}
	}
	return actNone
}

// SelectManagers shows the interactive checkbox menu when both in and out are
// TTYs; otherwise it falls back to the numbered line prompt. It returns the
// selected manager names in original order, or ErrSelectCanceled if the user
// cancels.
func SelectManagers(in, out *os.File, managers []buum.Manager) ([]string, error) {
	if !term.IsTerminal(int(in.Fd())) || !term.IsTerminal(int(out.Fd())) {
		return selectNumbered(in, out, managers)
	}
	return runSelectMenu(in, out, managers)
}

func runSelectMenu(in, out *os.File, managers []buum.Manager) ([]string, error) {
	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return selectNumbered(in, out, managers)
	}
	defer term.Restore(int(in.Fd()), oldState)

	names := make([]string, len(managers))
	for i, m := range managers {
		names[i] = m.Name
	}
	m := newSelectModel(names)

	// Redraw counts logical lines and moves the cursor up that many rows. This
	// assumes each rendered line occupies exactly one terminal row — true for the
	// short manager names and legend on a normal-width terminal. In a very narrow
	// or very short window a line could wrap or scroll, under-counting rows and
	// smearing the redraw; a scrolling viewport is intentionally out of scope
	// (YAGNI for the ~10-manager list).
	prevLines := 0
	draw := func() {
		if prevLines > 0 {
			fmt.Fprintf(out, "\r\033[%dA\033[J", prevLines)
		}
		frame := m.render()
		fmt.Fprint(out, frame)
		prevLines = strings.Count(frame, "\n")
	}
	draw()

	buf := make([]byte, 8)
	for {
		n, rerr := in.Read(buf)
		if rerr != nil {
			return nil, ErrSelectCanceled
		}
		switch apply(m, decodeKey(buf[:n])) {
		case actConfirm:
			fmt.Fprint(out, "\r\n")
			return m.selected(), nil
		case actCancel:
			fmt.Fprint(out, "\r\n")
			return nil, ErrSelectCanceled
		}
		draw()
	}
}

// PromptYesNo prints q and reads one line; only "y"/"yes" (case-insensitive)
// returns true. Any other input, or a read error, returns false.
func PromptYesNo(in io.Reader, out io.Writer, q string) bool {
	fmt.Fprint(out, q)
	line, _ := bufio.NewReader(in).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}
