package main

import (
	"fmt"
	"os"

	"github.com/kjk/7zd"
)

func usageAndExit() {
	fmt.Printf("usage: test file.7z")
	os.Exit(1)
}

func main() {
	if len(os.Args) != 2 {
		usageAndExit()
	}
	path := os.Args[1]
	_, err := lzmadec.NewArchive(path)
	if err != nil {
		fmt.Printf("lzmadec.NewArchive('%s') failed with '%s'\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("opened archive '%s'\n", path)
}
