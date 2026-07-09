package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("hello")
	if err := run(); err != nil {
		log.Fatal(err) // ok: log.Fatal is not os.Exit
	}
}

func run() error { return nil }
