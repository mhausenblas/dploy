package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "0.1.0"

func main() {
	beVerbose := flag.Bool("v", false, "Be verbose, defaults to false.")
	cmd := os.Args[1:]
	fmt.Printf("This is dploy version %s\n", version)
	fmt.Printf("Executing command %s\n", cmd)
	if *beVerbose {
		fmt.Printf("I will be verbose.")
	}
}
