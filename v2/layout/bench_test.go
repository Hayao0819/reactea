package layout

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Hayao0819/reactea/v2"
)

// panel draws a full box of text, the way a real meter or table would.
type panel struct {
	reactea.BasicComponent

	row string
}

func (p *panel) Render(ctx *reactea.Ctx) string {
	width, height := ctx.Size()

	rows := make([]string, height)
	for i := range rows {
		rows[i] = p.row[:min(len(p.row), width)]
	}

	return strings.Join(rows, "\n")
}

func dashboard(panes int, memoize bool) *Box {
	style := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

	columns := make([]Item, 0, panes)

	for range panes {
		var pane reactea.Component = Framed(style, &panel{row: strings.Repeat("x", 200)})

		if memoize {
			generation := 0
			pane = Memo(pane, func() any { return generation })
		}

		columns = append(columns, Grow(1, pane).Focusable())
	}

	return Column(
		Fixed(1, reactea.Text("header")),
		Grow(1, Row(columns...)),
		Fixed(1, reactea.Text("footer")),
	)
}

func BenchmarkRenderDashboard(b *testing.B) {
	app := reactea.New(dashboard(20, false), reactea.WithSize(200, 50))

	app.Init()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		app.View()
	}
}

func BenchmarkRenderDashboardMemoized(b *testing.B) {
	app := reactea.New(dashboard(20, true), reactea.WithSize(200, 50))

	app.Init()
	app.View()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		app.View()
	}
}

func BenchmarkUpdateDashboard(b *testing.B) {
	app := reactea.New(dashboard(20, false), reactea.WithSize(200, 50))

	app.Init()

	tick := struct{ tea.Msg }{}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		app.Update(tick)
	}
}
