package main

import (
	"github.com/Hayao0819/reactea"
	"github.com/Hayao0819/reactea/examples/quickstart/app"
)

func main() {
	// reactea.NewProgram initializes program with
	// "translation layer", so Reactea components work
	program := reactea.NewProgram(app.New())

	if _, err := program.Run(); err != nil {
		panic(err)
	}
}
