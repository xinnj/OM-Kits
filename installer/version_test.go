package main

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func TestVersionLabel(t *testing.T) {
	tests := []struct {
		version  string
		revision string
		want     string
	}{
		{version: "v2.4.0", revision: "", want: "v2.4.0"},
		{version: "", revision: "", want: "dev"},
		{version: "dev", revision: "", want: "dev"},
		{version: "dev", revision: "abcdef123456", want: "dev (abcdef1)"},
		{version: "dev", revision: "abc", want: "dev (abc)"},
		{version: "v2.4.0", revision: "abcdef123456", want: "v2.4.0"},
	}

	for _, tt := range tests {
		if got := versionLabel(tt.version, tt.revision); got != tt.want {
			t.Errorf("versionLabel(%q, %q) = %q, want %q", tt.version, tt.revision, got, tt.want)
		}
	}
}

func TestVersionStringReleaseOverride(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })

	version = "v2.4.0"
	if got := versionString(); got != "OM Kits Installer v2.4.0" {
		t.Fatalf("versionString() = %q, want %q", got, "OM Kits Installer v2.4.0")
	}
}

func TestBasicInfoPageShowsVersion(t *testing.T) {
	old := version
	t.Cleanup(func() { version = old })

	version = "v9.9.9"
	initFlexBasicInfo()

	if got := flexBasicInfo.GetItemCount(); got != 3 {
		t.Fatalf("basic info page has %d items, want 3", got)
	}
	versionView, ok := flexBasicInfo.GetItem(2).(*tview.TextView)
	if !ok {
		t.Fatalf("version item = %T, want *tview.TextView", flexBasicInfo.GetItem(2))
	}
	if got := versionView.GetText(true); !strings.HasSuffix(got, "OM Kits Installer v9.9.9") {
		t.Fatalf("version text = %q, want it to end with %q", got, "OM Kits Installer v9.9.9")
	}
	if got := versionView.GetBackgroundColor(); got != tview.Styles.PrimitiveBackgroundColor {
		t.Fatalf("version background = %v, want %v", got, tview.Styles.PrimitiveBackgroundColor)
	}

	// Draw the page and confirm the version is the last line of the help
	// block, one row above the bottom edge, with a blank row below it.
	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("init simulation screen: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 24)

	flexBasicInfo.SetRect(0, 0, 80, 24)
	flexBasicInfo.Draw(screen)
	screen.Show()

	cells, width, height := screen.GetContents()
	versionRow := screenRow(cells, width, height-2)
	if !strings.HasPrefix(versionRow, "OM Kits Installer v9.9.9") {
		t.Fatalf("row above bottom = %q, want it to start with the version", versionRow)
	}
	if lastRow := screenRow(cells, width, height-1); strings.TrimSpace(lastRow) != "" {
		t.Fatalf("last row = %q, want a blank row below the version", lastRow)
	}
}

func screenRow(cells []tcell.SimCell, width, y int) string {
	row := make([]rune, width)
	for x, cell := range cells[y*width : (y+1)*width] {
		if len(cell.Runes) > 0 {
			row[x] = cell.Runes[0]
		} else {
			row[x] = ' '
		}
	}
	return string(row)
}
