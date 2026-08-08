package main

import (
	"github.com/Hayao0819/reactea/v2"
	"github.com/Hayao0819/reactea/v2/examples/dynamicRoutes/app"
)

func main() {
	program := reactea.NewProgram(app.New())

	if _, err := program.Run(); err != nil {
		panic(err)
	}
}
