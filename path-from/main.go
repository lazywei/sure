package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func Parse() (string, string, error) {
	if len(os.Args) == 2 {
		base, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		target := os.Args[1]
		return base, target, nil
	} else if len(os.Args) == 3 {
		return os.Args[1], os.Args[2], nil
	} else {
		return "", "", fmt.Errorf("usage: path-from <base-path=cwd> <target-path>")
	}
}

func main() {
	base, target, err := Parse()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ret, err := filepath.Rel(base, target)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(ret)
}
