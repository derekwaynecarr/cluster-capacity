package main

import (
	"fmt"
	"os"

	"github.com/ingvagabund/cluster-capacity/cmd/genpod/app"
)

func main() {
	cmd := app.NewGenPodCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
