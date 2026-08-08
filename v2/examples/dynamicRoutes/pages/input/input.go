package input

import (
	"fmt"
	"strconv"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/examples/dynamicRoutes/data"
)

type Component struct {
	reactea.BasicComponent

	textinput textinput.Model

	ids []int
}

func New() *Component {
	var ids []int
	for id := range data.Players {
		ids = append(ids, id)
	}

	return &Component{
		textinput: textinput.New(),
		ids:       ids,
	}
}

func (c *Component) Init() tea.Cmd {
	return c.textinput.Focus()
}

func (c *Component) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEnter {
			// Validate input
			n, err := strconv.Atoi(c.textinput.Value())
			if err != nil {
				c.textinput.SetValue("Error")
				return nil
			}

			reactea.SetRoute(fmt.Sprintf("/players/%d", n))
			return nil
		}
	}

	var cmd tea.Cmd
	c.textinput, cmd = c.textinput.Update(msg)
	return cmd
}

func (c *Component) Render(int, int) string {
	return fmt.Sprintf("Found players with ids %v\nEnter player id: %s\nAnd press [ Enter ]", c.ids, c.textinput.View())
}
