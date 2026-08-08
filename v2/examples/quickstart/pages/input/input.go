package input

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type Component struct {
	reactea.BasicComponent

	SetText func(string)

	textinput textinput.Model
}

func New() *Component {
	return &Component{
		textinput: textinput.New(),
	}
}

func (c *Component) Init() tea.Cmd {
	return c.textinput.Focus()
}

func (c *Component) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.Code == tea.KeyEnter {
			// Lifted state power! Woohooo
			c.SetText(c.textinput.Value())

			reactea.SetRoute("/displayname")

			return nil
		}
	}

	var cmd tea.Cmd
	c.textinput, cmd = c.textinput.Update(msg)
	return cmd
}

// Here we are not using width and height, but you can!
// Using lipgloss styles for example
func (c *Component) Render(int, int) string {
	return fmt.Sprintf("Enter your name: %s\nAnd press [ Enter ]", c.textinput.View())
}
