package modal

import (
	"github.com/Hayao0819/reactea"
	tea "github.com/charmbracelet/bubbletea"
)

type ModalComponent[TReturn any] interface {
	reactea.Component

	initModal(chan<- ModalResult[TReturn], *Controller)
	Return(ModalResult[TReturn]) tea.Cmd
}
