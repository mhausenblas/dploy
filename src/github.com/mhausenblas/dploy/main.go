package main

import (
	"flag"
	"fmt"
	dploy "github.com/mhausenblas/dploy/dploy"
	"os"
)

var version = "0.2.0"
var initCmd = flag.NewFlagSet("init", flag.ExitOnError)
var locFlag = initCmd.String("location", ".", "Where to create the new DC/OS app.")
var verboseInitFlag = initCmd.Bool("v", false, "Print detailed information while initializing the new DC/OS app.")

func usage() {
	fmt.Println("Usage: dploy <command> [<args>]\n")
	fmt.Println("Available commands:")
	fmt.Println(" init ... creates a new app for you, that is, a `dploy.app` file with default values is created in -location")
	fmt.Println(" dryrun ... validates deployment of the app by checking if DC/OS cluster is valid, validates app specs, etc.")
}

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}
	fmt.Printf("This is dploy version %s\n", version)
	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		if initCmd.Parsed() {
			dploy.Init(*locFlag)
		}
	case "dryrun":
		dploy.DryRun()
	default:
		fmt.Printf("%q is not a valid command\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
