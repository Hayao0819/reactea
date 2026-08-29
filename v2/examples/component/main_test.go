package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Hayao0819/reactea/v2"
)

func TestBothParentsMountTheirChildren(t *testing.T) {
	byHand := &split{top: label("by hand"), bottom: &clock{}}

	root := &quitting{Wrapper: reactea.Wrap(byHand)}

	app := reactea.New(root, reactea.WithSize(24, 3))

	app.Start()
	app.Send(time.Unix(3661, 0).UTC())

	content := app.View().Content

	if !strings.Contains(content, "by hand (24 wide)") || !strings.Contains(content, "01:01:01") {
		t.Errorf("by hand:\n%s", content)
	}
}

func TestLayoutHoldsEveryChildToItsBox(t *testing.T) {
	app := reactea.New(laidOut(), reactea.WithSize(24, 5))

	app.Start()

	for _, line := range strings.Split(app.View().Content, "\n") {
		if len([]rune(line)) != 24 {
			t.Errorf("line %q is %d wide, want 24", line, len([]rune(line)))
		}
	}
}
