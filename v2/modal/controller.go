package modal

import (
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/Hayao0819/reactea/v2"
)

// Controller drives a modal flow. The flow itself is written imperatively in
// initFunc, which runs on its own goroutine (spawned by Run) and blocks on
// Show/Get until each modal returns a result. The bubbletea event loop drives
// the currently-shown modal through Update/Render.
//
// The tricky part is liveness: bubbletea only calls Update when a message
// arrives, so the flow goroutine can't just change state and expect the loop to
// notice. So when a modal returns, its Return call blocks until the flow
// goroutine has advanced to the next stable state (the next modal installed, or
// the flow ended). That handoff runs over the buffered `resume` channel, and it
// lets the very same Update pass observe the new state — no lost wakeups, no
// reliance on a follow-up message.
//
// All shared state is guarded by mu. cond lets Update block until the flow has
// something for it. Note that Update releases mu around modal.Update() so the
// flow goroutine can take the lock to advance while Return is blocking.
type Controller struct {
	reactea.BasicComponent

	initFunc   func(*Controller) func() tea.Cmd
	escapeFunc func() tea.Cmd // The flow's return value, run once when the flow ends

	mu   sync.Mutex
	cond *sync.Cond

	modal   reactea.Component
	initCmd tea.Cmd
	shown   bool // whether any modal has been shown yet (drives the resume handoff)
	ended   bool // set once the flow goroutine has returned
	done    bool // set once Update has handled the flow ending

	// resume hands control back from the flow goroutine to a modal's blocked
	// Return. Buffered(1) so the send can't be lost if Return hasn't parked yet;
	// modals run strictly one at a time, so it never holds more than one token.
	resume chan struct{}
}

func NewController(initFunc func(*Controller) func() tea.Cmd) *Controller {
	c := &Controller{
		initFunc: initFunc,
		resume:   make(chan struct{}, 1),
	}
	c.cond = sync.NewCond(&c.mu)

	return c
}

func (c *Controller) Init() tea.Cmd {
	// Rerender nudges the loop to run Update at least once so the first modal
	// renders (and a modal-less flow's escape still fires) without waiting for
	// input.
	return tea.Batch(c.Run(c.initFunc), reactea.Rerender)
}

func (c *Controller) Update(msg tea.Msg) tea.Cmd {
	c.mu.Lock()

	// Block the event loop until there is a modal to drive or the flow has
	// ended — a modal is a blocking overlay by design.
	for c.modal == nil && !c.ended {
		c.cond.Wait()
	}

	modal := c.modal

	var cmds []tea.Cmd

	if c.initCmd != nil {
		cmds = append(cmds, c.initCmd)
		c.initCmd = nil
	}

	c.mu.Unlock()

	// Drive the current modal WITHOUT holding the lock: its Return hands off to
	// the flow goroutine, which needs the lock to install the next modal or set
	// the escape. Return only unblocks once that handoff is complete, so the
	// state we re-read below already reflects the flow's next step.
	if modal != nil {
		cmds = append(cmds, modal.Update(msg))
	}

	c.mu.Lock()
	if c.ended && !c.done {
		c.done = true
		escapeFunc := c.escapeFunc
		c.escapeFunc = nil
		c.mu.Unlock()

		if escapeFunc != nil {
			cmds = append(cmds, escapeFunc())
		}
	} else {
		c.mu.Unlock()
	}

	return tea.Batch(cmds...)
}

// Destroy tears down the modal that is on screen. The flow goroutine destroys
// each modal as it advances, but a teardown while a modal is still shown never
// reaches that path.
func (c *Controller) Destroy() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.modal != nil {
		c.modal.Destroy()
		c.modal = nil
	}
}

func (c *Controller) Render(width, height int) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.modal != nil {
		return c.modal.Render(width, height)
	}

	return ""
}

func (c *Controller) Run(f func(*Controller) func() tea.Cmd) tea.Cmd {
	go func() {
		escapeFunc := f(c)

		c.mu.Lock()
		c.escapeFunc = escapeFunc
		c.ended = true
		c.cond.Broadcast()
		c.mu.Unlock()

		// Release the last modal's blocked Return so its Update pass can observe
		// the flow ending and fire the escape.
		c.resume <- struct{}{}
	}()

	return nil
}
