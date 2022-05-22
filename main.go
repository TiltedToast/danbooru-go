package main

import (
	"fmt"
	"os"
)


type inputOptions struct {
	tag string
	outputDir string
	safe bool
	risky bool
	explicit bool
}


func main() {
	args := os.Args[1:]
	fmt.Println(args)
}