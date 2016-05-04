package main

import (
	"flag"
	"fmt"
	dploy "github.com/mhausenblas/dploy/lib"
	"os"
	"strings"
)

const (
	BANNER = `    .___        .__                  
  __| _/______  |  |    ____  ___.__.
 / __ | \____ \ |  |   /  _ \<   |  |
/ /_/ | |  |_> >|  |__(  <_> )\___  |
\____ | |   __/ |____/ \____/ / ____|
     \/ |__|                  \/     
`
	VERSION = "0.5.4"
)

var (
	cmd     string
	workdir = "./"
	help    bool
)

func about() {
	fmt.Fprint(os.Stderr, BANNER)
	fmt.Fprint(os.Stderr, fmt.Sprintf("This is dploy version %s, using workdir %s\n", VERSION, workdir))
	fmt.Fprint(os.Stderr, fmt.Sprintf("Please visit http://dploy.sh to learn more about me,\n"))
	fmt.Fprint(os.Stderr, fmt.Sprintf("report issues and also how to contribute to this project.\n"))
	fmt.Fprint(os.Stderr, strings.Repeat("=", 57), "\n")
}

func init() {
	// flag.BoolVar(&help, "help", false, "print help for a command and exit")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: dploy <command> [workdir] [command-args]\n")
		fmt.Fprint(os.Stderr, "\nValid values for `command` are:\n")
		fmt.Fprint(os.Stderr, "\tinit ... creates a new app descriptor and inits `specs/`\n")
		fmt.Fprint(os.Stderr, "\tdryrun ... validates app deployment using Marathon API\n")
		fmt.Fprint(os.Stderr, "\trun ... launches the app using `dploy.app` and the content of `specs/`\n")
		fmt.Fprint(os.Stderr, "\tdestroy ... tears down the app\n")
		fmt.Fprint(os.Stderr, "\tls ... lists the app's resources\n")
		fmt.Fprint(os.Stderr, "\nThe `workdir` parameters defaults to `./`, that is, the current directory.\n")
		// fmt.Fprint(os.Stderr, "\nValid values for `command-args` are:\n")
		// flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		about()
		flag.Usage()
		os.Exit(1)
	} else {
		cmd = flag.Args()[0]
		if flag.NArg() > 1 {
			workdir = flag.Args()[1]
		}
	}
}

func main() {
	about()
	switch cmd {
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
		fmt.Fprint(os.Stderr, flag.Args()[0], " is not a valid command\n")
		flag.Usage()
		os.Exit(2)
	}
}
