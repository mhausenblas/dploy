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
	VERSION = "0.7.0"
)

var (
	cmd string
	// global arguments:
	workspace string
	all       bool
	// command-specific arguments:
	pid       string
	instances int
)

func about() {
	fmt.Fprint(os.Stderr, BANNER)
	fmt.Fprint(os.Stderr, fmt.Sprintf("This is dploy version %s, using workspace [%s]\n", VERSION, workspace))
	fmt.Fprint(os.Stderr, fmt.Sprintf("Please visit http://dploy.sh to learn more about me,\n"))
	fmt.Fprint(os.Stderr, fmt.Sprintf("report issues and also how to contribute to this project.\n"))
	fmt.Fprint(os.Stderr, strings.Repeat("=", 57), "\n")
}

func init() {
	cwd, _ := os.Getwd()
	flag.StringVar(&workspace, "workspace", cwd, "[GLOBAL] directory in which to operate")
	flag.StringVar(&workspace, "w", cwd, "[GLOBAL] directory in which to operate (shorthand)")
	flag.BoolVar(&all, "all", false, "[GLOBAL] output all available data, semantics are command dependent")
	flag.BoolVar(&all, "a", false, "[GLOBAL] output all available data, semantics are command dependent (shorthand)")
	flag.StringVar(&pid, "pid", "", "[SCALE] target the µS with pid")
	flag.IntVar(&instances, "instances", 0, "[SCALE] set the number of instances")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: dploy [args] <command>\n")
		fmt.Fprint(os.Stderr, "\nThe following commands are available:\n")
		fmt.Fprint(os.Stderr, "\tinit\t... creates a new app descriptor and inits `specs/`\n")
		fmt.Fprint(os.Stderr, "\tdryrun\t... validates app deployment using Marathon API\n")
		fmt.Fprint(os.Stderr, "\trun\t... launches the app using `dploy.app` and the content of `specs/`\n")
		fmt.Fprint(os.Stderr, "\tdestroy\t... tears down the app\n")
		fmt.Fprint(os.Stderr, "\tls\t... lists the app's resources\n")
		fmt.Fprint(os.Stderr, "\tps\t... lists runtime properties of the app\n")
		fmt.Fprint(os.Stderr, "\tscale\t... scales a µS in the app\n")
		fmt.Fprint(os.Stderr, "\nValid (optional) arguments are:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		about()
		flag.Usage()
		os.Exit(1)
	} else {
		cmd = flag.Args()[0]
	}
}

func main() {
	about()
	success := false
	switch cmd {
	case "init":
		success = dploy.Init(workspace, all)
	case "dryrun":
		success = dploy.DryRun(workspace, all)
	case "run":
		success = dploy.Run(workspace, all)
	case "destroy":
		success = dploy.Destroy(workspace, all)
	case "ls":
		success = dploy.ListResources(workspace, all)
	case "ps":
		success = dploy.ListRuntimeProperties(workspace, all)
	case "scale":
		success = dploy.Scale(workspace, all, pid, instances)
	default:
		fmt.Fprint(os.Stderr, flag.Args()[0], " is not a valid dploy command\n")
		flag.Usage()
		os.Exit(2)
	}
	if success {
		os.Exit(0)
	} else {
		os.Exit(3)
	}
}
