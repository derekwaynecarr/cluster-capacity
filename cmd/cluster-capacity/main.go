package main

import (
	"fmt"
	"os"

	"github.com/ingvagabund/cluster-capacity/cmd/cluster-capacity/app"
)

func main() {
	cmd := app.NewClusterCapacityCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
