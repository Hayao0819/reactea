package reactea

import "testing"

func TestOptions(t *testing.T) {
	t.Run("WithRoute", func(t *testing.T) {
		root := &mockComponent[struct{}]{}

		NewProgram(root, WithRoute("/testRoute"))

		if CurrentRoute() != "/testRoute" {
			t.Errorf("expected current route \"/testRoute\", but got \"%s\"", CurrentRoute())
		}
	})

	t.Run("WithRouteEmpty", func(t *testing.T) {
		// An empty route must be ignored, not indexed (route[0] would panic).
		root := &mockComponent[struct{}]{}

		NewProgram(root, WithRoute(""))

		if CurrentRoute() != "/" {
			t.Errorf("empty route should be ignored, expected \"/\", got \"%s\"", CurrentRoute())
		}
	})

	t.Run("WithRouteNonRoot", func(t *testing.T) {
		// A route without a leading '/' is invalid and must be rejected,
		// matching SetRoute's own validation.
		root := &mockComponent[struct{}]{}

		NewProgram(root, WithRoute("noLeadingSlash"))

		if CurrentRoute() != "/" {
			t.Errorf("non-root route should be ignored, expected \"/\", got \"%s\"", CurrentRoute())
		}
	})

	t.Run("WithoutInput", func(t *testing.T) {
		root := &mockComponent[struct{}]{
			renderFunc: func(c Component, s *struct{}, width, height int) string {
				return "test passed"
			},
		}

		program := NewProgram(root, WithoutInput())

		go program.Quit()

		if _, err := program.Run(); err != nil {
			t.Fatal(err)
		}
	})
}
