package modal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/londek/reactea"
)

type Modal[T any] struct {
	ch chan<- ModalResult[T]
	c  *Controller
}

//lint:ignore U1000 This function is used, but through interface
func (modal *Modal[T]) initModal(resultChan chan<- ModalResult[T], controller *Controller) {
	modal.ch = resultChan
	modal.c = controller
}

func (modal *Modal[T]) Return(result ModalResult[T]) tea.Cmd {
	// Hand the result to the flow goroutine (parked in Show/Get), then wait for
	// it to advance to the next stable state before letting this Update pass
	// continue. See Controller for why this handshake is needed for liveness.
	modal.ch <- result
	<-modal.c.resume

	return reactea.Rerender
}

func (modal *Modal[T]) Ok(result T) tea.Cmd {
	return modal.Return(Ok(result))
}

func (modal *Modal[T]) Error(err error) tea.Cmd {
	return modal.Return(Error[T](err))
}

func Show[T any](c *Controller, modal ModalComponent[T]) ModalResult[T] {
	resultChan := make(chan ModalResult[T])

	modal.initModal(resultChan, c)

	c.mu.Lock()
	prevShown := c.shown
	c.shown = true
	c.modal = modal
	c.initCmd = modal.Init()
	c.cond.Broadcast()
	c.mu.Unlock()

	// If a previous modal's Return is waiting for the flow to advance, release
	// it now that this modal is installed.
	if prevShown {
		c.resume <- struct{}{}
	}

	// Block this flow goroutine until the modal calls Return.
	result := <-resultChan

	// Tear the modal down deterministically here (on the flow goroutine), while
	// holding the lock, so Render can never observe or destroy the wrong modal.
	c.mu.Lock()
	if c.modal == modal {
		modal.Destroy()
		c.modal = nil
	}
	c.mu.Unlock()

	return result
}

func Get[T any](c *Controller, modal ModalComponent[T]) T {
	return Show(c, modal).Return
}
