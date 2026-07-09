package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello")
	os.Exit(1) // want `direct call to os.Exit in main function`
}
