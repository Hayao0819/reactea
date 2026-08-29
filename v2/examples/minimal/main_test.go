package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

func TestCounterCounts(t *testing.T) {
	app := reactea.New(&counter{}, reactea.WithSize(40, 1))

	app.Start()
	app.Send(tea.KeyPressMsg{Code: '+', Text: "+"}, tea.KeyPressMsg{Code: '+', Text: "+"})

	if got := app.View().Content; got != "count 2 — + to add, q to quit" {
		t.Errorf("content = %q", got)
	}
}
