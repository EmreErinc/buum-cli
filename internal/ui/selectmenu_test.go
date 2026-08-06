package ui

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func TestNewSelectModelAllChecked(t *testing.T) {
	m := newSelectModel([]string{"brew", "npm", "cargo"})
	if got := m.selected(); !reflect.DeepEqual(got, []string{"brew", "npm", "cargo"}) {
		t.Fatalf("expected all checked, got %v", got)
	}
	if sel, total := m.counts(); sel != 3 || total != 3 {
		t.Fatalf("counts got %d/%d want 3/3", sel, total)
	}
}

func TestToggleUnderCursor(t *testing.T) {
	m := newSelectModel([]string{"brew", "npm", "cargo"})
	m.moveDown() // cursor -> npm
	m.toggle()
	if got := m.selected(); !reflect.DeepEqual(got, []string{"brew", "cargo"}) {
		t.Fatalf("got %v", got)
	}
}

func TestMoveWraps(t *testing.T) {
	m := newSelectModel([]string{"a", "b", "c"})
	m.moveUp() // wrap to last
	if m.cursor != 2 {
		t.Fatalf("moveUp from 0 should wrap to 2, got %d", m.cursor)
	}
	m.moveDown() // wrap to first
	if m.cursor != 0 {
		t.Fatalf("moveDown from 2 should wrap to 0, got %d", m.cursor)
	}
}

func TestToggleAllSmart(t *testing.T) {
	m := newSelectModel([]string{"a", "b"}) // all checked
	m.toggleAll()                           // -> all unchecked
	if sel, _ := m.counts(); sel != 0 {
		t.Fatalf("toggleAll on all-checked should clear, got %d", sel)
	}
	m.toggleAll() // -> all checked
	if sel, _ := m.counts(); sel != 2 {
		t.Fatalf("toggleAll on empty should check all, got %d", sel)
	}
	// mixed -> fill all
	m2 := newSelectModel([]string{"a", "b"})
	m2.toggle() // uncheck 'a' (cursor at 0) -> mixed
	m2.toggleAll()
	if sel, _ := m2.counts(); sel != 2 {
		t.Fatalf("toggleAll on mixed should fill all, got %d", sel)
	}
}

func TestFilterVisibleAndSelectionPersists(t *testing.T) {
	m := newSelectModel([]string{"brew", "cargo", "rustup"})
	m.setFilter("ca") // only cargo visible
	if vis := m.visible(); len(vis) != 1 || m.items[vis[0]] != "cargo" {
		t.Fatalf("filter 'ca' visible wrong: %v", vis)
	}
	m.toggle() // uncheck cargo (cursor at 0 of visible)
	m.setFilter("")
	if got := m.selected(); !reflect.DeepEqual(got, []string{"brew", "rustup"}) {
		t.Fatalf("selection must persist across filter, got %v", got)
	}
}

func TestSelectedOriginalOrder(t *testing.T) {
	m := newSelectModel([]string{"z", "a", "m"})
	if got := m.selected(); !reflect.DeepEqual(got, []string{"z", "a", "m"}) {
		t.Fatalf("selected must keep original order, got %v", got)
	}
}

func TestDecodeKey(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want key
		r    rune
	}{
		{"up", []byte{0x1b, '[', 'A'}, keyUp, 0},
		{"down", []byte{0x1b, '[', 'B'}, keyDown, 0},
		{"lone esc", []byte{0x1b}, keyEsc, 0},
		{"enter cr", []byte{0x0d}, keyEnter, 0},
		{"enter lf", []byte{0x0a}, keyEnter, 0},
		{"space", []byte{0x20}, keySpace, 0},
		{"backspace del", []byte{0x7f}, keyBackspace, 0},
		{"ctrl-c", []byte{0x03}, keyCtrlC, 0},
		{"printable a", []byte{'a'}, keyPrintable, 'a'},
		{"printable slash", []byte{'/'}, keyPrintable, '/'},
		{"empty", []byte{}, keyNone, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := decodeKey(c.in)
			if got.kind != c.want {
				t.Fatalf("kind: got %d want %d", got.kind, c.want)
			}
			if c.r != 0 && got.r != c.r {
				t.Fatalf("rune: got %q want %q", got.r, c.r)
			}
		})
	}
}

func TestApplyListMode(t *testing.T) {
	m := newSelectModel([]string{"brew", "npm"})

	if a := apply(m, keyEvent{kind: keyDown}); a != actNone || m.cursor != 1 {
		t.Fatalf("down: action %d cursor %d", a, m.cursor)
	}
	if a := apply(m, keyEvent{kind: keySpace}); a != actNone {
		t.Fatalf("space action %d", a)
	}
	if sel, _ := m.counts(); sel != 1 { // npm unchecked
		t.Fatalf("space should toggle, sel=%d", sel)
	}
	if a := apply(m, keyEvent{kind: keyPrintable, r: 'a'}); a != actNone {
		t.Fatalf("'a' action %d", a)
	}
	if sel, _ := m.counts(); sel != 2 { // mixed -> fill all (spec: else check all)
		t.Fatalf("'a' toggleAll sel=%d want 2", sel)
	}
	if a := apply(m, keyEvent{kind: keyEnter}); a != actConfirm {
		t.Fatalf("enter should confirm, got %d", a)
	}
	if a := apply(m, keyEvent{kind: keyPrintable, r: 'q'}); a != actCancel {
		t.Fatalf("'q' should cancel, got %d", a)
	}
	if a := apply(m, keyEvent{kind: keyEsc}); a != actCancel {
		t.Fatalf("esc should cancel, got %d", a)
	}
}

func TestApplyEntersFilterMode(t *testing.T) {
	m := newSelectModel([]string{"brew", "cargo"})
	apply(m, keyEvent{kind: keyPrintable, r: '/'})
	if !m.filtering {
		t.Fatal("'/' should enter filter mode")
	}
	apply(m, keyEvent{kind: keyPrintable, r: 'c'})
	apply(m, keyEvent{kind: keyPrintable, r: 'a'})
	if m.filter != "ca" {
		t.Fatalf("filter got %q want 'ca'", m.filter)
	}
	if vis := m.visible(); len(vis) != 1 {
		t.Fatalf("filtered visible=%d want 1", len(vis))
	}
	apply(m, keyEvent{kind: keyBackspace})
	if m.filter != "c" {
		t.Fatalf("backspace got %q want 'c'", m.filter)
	}
	apply(m, keyEvent{kind: keyEnter}) // exit filter mode, keep filter
	if m.filtering {
		t.Fatal("enter should exit filter mode")
	}
	if m.filter != "c" {
		t.Fatalf("filter should persist after exit, got %q", m.filter)
	}
}

func TestApplyFilterModeQIsLiteral(t *testing.T) {
	m := newSelectModel([]string{"q-tool"})
	apply(m, keyEvent{kind: keyPrintable, r: '/'})
	if a := apply(m, keyEvent{kind: keyPrintable, r: 'q'}); a != actNone {
		t.Fatalf("'q' in filter mode must be literal, got action %d", a)
	}
	if m.filter != "q" {
		t.Fatalf("filter got %q want 'q'", m.filter)
	}
}

func TestRenderListFrame(t *testing.T) {
	m := newSelectModel([]string{"brew", "npm"})
	m.moveDown() // cursor on npm
	m.toggle()   // uncheck npm
	out := m.render()
	if !strings.Contains(out, "1 of 2 selected") {
		t.Errorf("missing count header:\n%s", out)
	}
	if !strings.Contains(out, "  [x] brew") {
		t.Errorf("brew should be checked, not cursored:\n%s", out)
	}
	if !strings.Contains(out, "> [ ] npm") {
		t.Errorf("npm should be cursored + unchecked:\n%s", out)
	}
	if !strings.Contains(out, "space toggle") {
		t.Errorf("missing legend line:\n%s", out)
	}
}

func TestRenderFilterEmptyState(t *testing.T) {
	m := newSelectModel([]string{"brew"})
	m.filtering = true
	m.setFilter("zzz")
	out := m.render()
	if !strings.Contains(out, "(no matches)") {
		t.Errorf("expected empty state:\n%s", out)
	}
	if !strings.Contains(out, "/ zzz") {
		t.Errorf("expected filter header:\n%s", out)
	}
}

func TestPromptYesNo(t *testing.T) {
	cases := map[string]bool{"y\n": true, "Y\n": true, "yes\n": true, "n\n": false, "\n": false, "nope\n": false}
	for in, want := range cases {
		got := PromptYesNo(strings.NewReader(in), &bytes.Buffer{}, "Save? ")
		if got != want {
			t.Errorf("input %q: got %v want %v", in, got, want)
		}
	}
}
