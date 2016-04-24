package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"os"
)

var version = "0.1.0"
var initCmd = flag.NewFlagSet("init", flag.ExitOnError)
var locFlag = initCmd.String("location", ".", "Where to create the new DC/OS app.")
var verboseInitFlag = initCmd.Bool("v", false, "Print detailed information while initializing the new DC/OS app.")

func deploy_usage() {
	fmt.Println("Usage: dploy <command> [<args>]\n")
	fmt.Println("Available commands:")
	fmt.Println(" init ... creates a new app for you, that is, a `dploy.app` file with default values is created in -location")
}

func deploy_init() {
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", *locFlag)
}

func main() {
	if len(os.Args) == 1 {
		deploy_usage()
		os.Exit(1)
	}
	fmt.Printf("This is dploy version %s\n", version)
	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(2)
	}
	if initCmd.Parsed() {
		deploy_init()
	}
}
