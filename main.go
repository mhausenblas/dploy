package main

import (
	"flag"
	"log"
	"os"
)

var version = "0.1.0"
var logger = log.New(os.Stdout, "[DPLOY]: ", log.Ldate|log.Ltime)
var verboseReporting = false

func deploy_init() {
	logger.Printf("In init")
}

func main() {
	beVerbose := flag.Bool("v", false, "Be verbose, defaults to false.")
	cmd := os.Args[1:1]
	logger.Printf("This is dploy version %s\n", version)
	logger.Printf("Executing command %s\n", cmd)
	if *beVerbose {
		verboseReporting = true
	}
	deploy_init()
}
