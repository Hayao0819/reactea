package modal

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

type ModalComponent[TReturn any] interface {
	reactea.Component

	initModal(chan<- ModalResult[TReturn], *Controller)
	Return(ModalResult[TReturn]) tea.Cmd
}
