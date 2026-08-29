// Command minimal is the smallest complete reactea program: one component with
// state, one key binding, and a line to run it.
package main

import (
	"fmt"
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type counter struct {
	reactea.BasicComponent

	count int
}

func (c *counter) Update(_ *reactea.Ctx, msg tea.Msg) tea.Cmd {
	switch {
	case reactea.Key(msg, "q", "ctrl+c"):
		return tea.Quit
	case reactea.Key(msg, "+"):
		c.count++
	}

	return nil
}

func (c *counter) Render(*reactea.Ctx) string {
	return fmt.Sprintf("count %d — + to add, q to quit", c.count)
}

func main() {
	if err := reactea.New(&counter{}).Run(); err != nil {
		log.Fatal(err)
	}
}
