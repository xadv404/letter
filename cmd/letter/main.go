package main

import (
	"fmt"
	"os"

	"github.com/xadv404/letter/internal/dashboard"
)

func main() {
	if err := dashboard.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
