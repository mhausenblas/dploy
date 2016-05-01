package main

import (
	"fmt"
	dploy "github.com/mhausenblas/dploy/lib"
	"os"
	"strings"
)

var (
	version = "0.5.1"
	workdir = "./"
)

func usage() {
	about()
	fmt.Println("\nUsage: dploy <command> [workdir] [command-args]\n")
	fmt.Println("Valid values for `command` are:")
	fmt.Println("\tinit ... creates a new app descriptor and inits `specs/`")
	fmt.Println("\tdryrun ... validates app deployment using Marathon API")
	fmt.Println("\trun ... launches the app using `dploy.app` and the content of `specs/`")
	fmt.Println("\tdestroy ... tears down the app")
	fmt.Println("\tls ... lists the app's resources")
	fmt.Println("Note that `workdir` defaults to `./`, that is, the current directory.")
}

func about() {
	fmt.Printf("This is dploy version %s\n", version)
	fmt.Printf("\tUsing workdir: %s\n", workdir)
	fmt.Println("\tPlease visit http://dploy.sh to learn more about me,")
	fmt.Println("\treport issues and also how to contribute to this project.")
	fmt.Println(strings.Repeat("=", 80))
}

func main() {
	if len(os.Args) == 1 {
		usage()
		os.Exit(1)
	} else {
		if len(os.Args) > 2 {
			workdir = os.Args[2]
		}
	}
	about()

	switch os.Args[1] {
	case "init":
		dploy.Init(workdir)
	case "dryrun":
		dploy.DryRun(workdir)
	case "run":
		dploy.Run(workdir)
	case "destroy":
		dploy.Destroy(workdir)
	case "ls":
		dploy.ListResources(workdir)
	default:
		fmt.Printf("%q is not a valid command\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}
