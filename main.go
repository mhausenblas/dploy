package main

import (
	"flag"
	"fmt"
	dploy "github.com/mhausenblas/dploy/lib"
	"os"
	"strings"
)

var version = "0.5.0"
var initCmd = flag.NewFlagSet("init", flag.ExitOnError)
var locFlag = initCmd.String("location", ".", "Where to create the new DC/OS app.")

func usage() {
	about()
	fmt.Println("\nUsage: dploy <command> [<args>]\n")
	fmt.Println("Available commands:")
	fmt.Println("\tinit ... creates a new app for you, that is, a `dploy.app` file with default values is created in -location")
	fmt.Println("\tdryrun ... validates deployment of the app by checking if DC/OS cluster is valid, validates app specs, etc.")
	fmt.Println("\trun ... launches the app using `dploy.app` and the content of `specs/`")
	fmt.Println("\tdestroy ... tears down the app")
	fmt.Println("\tls ... lists the resources of the app")
}

func about() {
	fmt.Printf("This is dploy version %s\n", version)
	fmt.Println("\tPlease visit http://dploy.sh to learn more about me,")
	fmt.Println("\treport issues and also how to contribute to this project.")
	fmt.Println(strings.Repeat("=", 80))
}

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	}
	about()
	switch os.Args[1] {
	case "init":
		initCmd.Parse(os.Args[2:])
		if initCmd.Parsed() {
			dploy.Init(*locFlag)
		}
	case "dryrun":
		dploy.DryRun()
	case "run":
		dploy.Run()
	case "destroy":
		dploy.Destroy()
	case "ls":
		dploy.ListResources()
	default:
		fmt.Printf("%q is not a valid command\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
