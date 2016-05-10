package dploy

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	tw "github.com/olekukonko/tablewriter"
	yaml "gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	ENV_VAR_DPLOY_LOGLEVEL     string        = "DPLOY_LOGLEVEL"
	ENV_VAR_DPLOY_EXAMPLES     string        = "DPLOY_EXAMPLES"
	DEFAULT_DEPLOY_WAIT_TIME   time.Duration = 10
	APP_DESCRIPTOR_FILENAME    string        = "dploy.app"
	DEFAULT_MARATHON_URL       string        = "http://localhost:8080"
	DEFAULT_APP_NAME           string        = "CHANGEME"
	MARATHON_APP_SPEC_DIR      string        = "specs/"
	MARATHON_APP_SPEC_EXT      string        = ".json"
	MARATHON_LABEL             string        = "DPLOY"
	MARATHON_OBSERVER_TEMPLATE string        = "https://raw.githubusercontent.com/mhausenblas/dploy/master/observer/observer.json"
	RESOURCETYPE_PLATFORM      string        = "platform"
	RESOURCETYPE_APP           string        = "app"
	RESOURCETYPE_GROUP         string        = "group"
	CMD_TRUNCATE               int           = 17
	EXAMPLE_HELLO_WORLD        string        = "https://raw.githubusercontent.com/mhausenblas/dploy/master/examples/helloworld.json"
	EXAMPLE_BUZZ               string        = "https://raw.githubusercontent.com/mhausenblas/dploy/master/examples/buzz/buzz.json"
	USER_MSG_SUCCESS           string        = "🙌"
	USER_MSG_PROBLEM           string        = "🙁"
	USER_MSG_INFO              string        = "🗣"
	SYSTEM_MSG_ONLINE          string        = "online \t💚"
	SYSTEM_MSG_OFFLINE         string        = "offline\t💔"
)

// DployApp is the dploy application deployment descriptor, in short: app descriptor.
// It defines the connection to the target DC/OS cluster as well as the app properties.
type DployApp struct {
	MarathonURL string `yaml:"marathon_url"`
	AppName     string `yaml:"app_name"`
	RepoURL     string `yaml:"repo_url,omitempty"`
	PublicNode  string `yaml:"public_node,omitempty"`
	PAToken     string `yaml:"pat,omitempty"`
}

// Init creates an app descriptor (dploy.app) and the `specs/` directory
// in the workdir specified as well as copies in example app specs.
// For example:
//  dploy.Init("../.")
func Init(workdir string, showAll bool) bool {
	setLogLevel()
	ensureWorkDir(workdir)
	fmt.Printf("%s\tInitializing your app ...\n", USER_MSG_INFO)
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", workdir)
	appDescriptor := DployApp{}
	appDescriptor.MarathonURL = DEFAULT_MARATHON_URL
	appDescriptor.AppName = DEFAULT_APP_NAME
	d, err := yaml.Marshal(&appDescriptor)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "init"}).Error("Failed to serialize dploy app descriptor due to error: ", err)
		return false
	}
	log.WithFields(log.Fields{"cmd": "init"}).Debug("Trying to create app descriptor ", APP_DESCRIPTOR_FILENAME, " with following content:\n", string(d))
	appDescriptorLocation, _ := filepath.Abs(filepath.Join(workdir, APP_DESCRIPTOR_FILENAME))
	writeData(appDescriptorLocation, string(d))
	specsDir, _ := filepath.Abs(filepath.Join(workdir, MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		os.Mkdir(specsDir, 0755)
		log.WithFields(log.Fields{"cmd": "init"}).Info("Created ", specsDir)
	}
	fmt.Printf("%s\tDone initializing your app:\n", USER_MSG_SUCCESS)
	fmt.Printf("\t\tSet up app descriptor in %s\n", appDescriptorLocation)
	fmt.Printf("\t\tCreated app spec directory %s\n", specsDir)
	ex := os.Getenv(ENV_VAR_DPLOY_EXAMPLES)
	switch strings.ToLower(ex) {
	case "all":
		Download(EXAMPLE_HELLO_WORLD, specsDir)
		Download(EXAMPLE_BUZZ, specsDir)
		fmt.Printf("\t\tInitialized app spec directory with some examples\n")
	case "buzz":
		Download(EXAMPLE_BUZZ, specsDir)
		fmt.Printf("\t\tInitialized app spec directory with the buzz example\n")
	default:
	}
	fmt.Printf("%s\tNow it's time to edit the app descriptor and adapt or add Marathon app specs.\n", USER_MSG_INFO)
	fmt.Printf("\tNext, you can run `dploy dryrun`\n")
	return true
}

// DryRun validates the app descriptor by checking if Marathon is reachable and also
// checks if the app spec directory is present, incl. at least one Marathon app spec.
func DryRun(workdir string, showAll bool) bool {
	setLogLevel()
	fmt.Printf("%s\tKicking the tires! Checking Marathon connection, descriptor and app specs ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor(workdir)
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "dryrun"}).Error("Failed to connect to Marathon due to error ", err)
		return false
	}
	info := marathonGetInfo(*marathonURL)
	fmt.Printf("%s\tFound DC/OS Marathon instance\n", USER_MSG_SUCCESS)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" name: ", info.Name)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" version: ", info.Version)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" leader: ", info.Leader)

	specsDir, _ := filepath.Abs(filepath.Join(workdir, MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		fmt.Printf("%s\tDidn't find app spec dir, expecting it in %s\n", USER_MSG_PROBLEM, specsDir)
		fmt.Printf("%s\tTry `dploy init` here first.\n", USER_MSG_INFO)
		return false
	} else {
		appDescriptor := readAppDescriptor(workdir)
		if strings.HasPrefix(appDescriptor.MarathonURL, "http") {
			fmt.Printf("%s\tFound an app descriptor\n", USER_MSG_SUCCESS)
			if appSpecs := getAppSpecs(workdir); len(appSpecs) > 0 {
				fmt.Printf("%s\tFound %d app spec(s) to deploy\n", USER_MSG_SUCCESS, len(appSpecs))
			} else {
				fmt.Printf("%s\tDidn't find any app specs in %s \n", USER_MSG_PROBLEM, MARATHON_APP_SPEC_DIR)
				return false
			}
		} else {
			fmt.Printf("%s\tDidn't find an app descriptor (%s) in current directory\n", USER_MSG_PROBLEM, APP_DESCRIPTOR_FILENAME)
			return false
		}
	}
	// check for optional push-to-deploy info,
	// i.e. both a GitHub repo URL and a public node
	// have been set in the `dploy.app` file
	if appDescriptor.RepoURL != "" && appDescriptor.PublicNode != "" && appDescriptor.PAToken != "" {
		fmt.Printf("%s\tFound stuff I need for push-to-deploy:\n", USER_MSG_SUCCESS)
		fmt.Printf("\tGitHub repo %s\n", appDescriptor.RepoURL)
		fmt.Printf("\tPublic node %s\n", appDescriptor.PublicNode)
		fmt.Printf("\tGitHub personal access token %s\n", strings.Repeat("*", len(appDescriptor.PAToken)))
	}
	fmt.Printf("%s\tNow you can use `dploy ls` to list resources of your app\n", USER_MSG_INFO)
	fmt.Printf("\tor `dploy run` to launch it via Marathon.\n")
	return true
}

// Run launches the app as defined in the descriptor and the app specs.
// It scans the `specs/` directory for Marathon app specs and launches them using the Marathon API.
func Run(workdir string, showAll bool) bool {
	setLogLevel()
	fmt.Printf("%s\tOK, let's rock and roll! Trying to launch your app ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor(workdir)
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "run"}).Error("Failed to connect to Marathon due to error ", err)
		return false
	}
	fmt.Printf("%s\tWorking\n", USER_MSG_INFO)
	go showSpinner(100 * time.Millisecond)
	marathonCreateApps(*marathonURL, appDescriptor.AppName, workdir)
	hideSpinner()
	fmt.Printf("%s\tLaunched your app!\n", USER_MSG_SUCCESS)
	// check and if configured launched the push-to-deploy
	// support via the observer service:
	launchObserver(appDescriptor, workdir)
	fmt.Printf("%s\tNow you can use `dploy ps` to list processes\n", USER_MSG_INFO)
	fmt.Printf("\tor `dploy destroy` to tear down the app again.\n")
	return true
}

// Destroy tears down the app.
// It scans the `specs/` directory for Marathon app specs and deletes apps using the Marathon API.
func Destroy(workdir string, showAll bool) bool {
	setLogLevel()
	fmt.Printf("%s\tSeems you wanna get rid of your app. OK, gonna try and tear it down now ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor(workdir)
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "destroy"}).Error("Failed to connect to Marathon due to error ", err)
		return false
	}
	fmt.Printf("%s\tWorking\n", USER_MSG_INFO)
	go showSpinner(100 * time.Millisecond)
	marathonDeleteApps(*marathonURL, appDescriptor.AppName, workdir)
	hideSpinner()
	fmt.Printf("%s\tDestroyed your app!\n", USER_MSG_SUCCESS)
	return true
}

// ListResources lists the resource definitions of the app.
func ListResources(workdir string, showAll bool) bool {
	setLogLevel()
	appDescriptor := readAppDescriptor(workdir)
	specsDir, _ := filepath.Abs(filepath.Join(workdir, MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		fmt.Printf("%s\tDidn't find app spec dir, expecting it in %s\n", USER_MSG_PROBLEM, specsDir)
		fmt.Printf("%s\tTry `dploy init` here first.\n", USER_MSG_INFO)
		return false
	} else {
		if strings.HasPrefix(appDescriptor.MarathonURL, "http") {
			renderAppResources(appDescriptor, workdir)
		} else {
			fmt.Printf("%s\tDidn't find an app descriptor (%s) in current directory\n", USER_MSG_PROBLEM, APP_DESCRIPTOR_FILENAME)
			return false
		}
	}
	return true
}

// ListRuntimeProperties lists runtime properties of the app.
func ListRuntimeProperties(workdir string, showAll bool) bool {
	setLogLevel()
	appDescriptor := readAppDescriptor(workdir)
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "ps"}).Error("Failed to connect to Marathon due to error ", err)
		return false
	}
	myApps := marathonAppRuntime(*marathonURL, appDescriptor.AppName)
	table := tw.NewWriter(os.Stdout)
	if showAll {
		table.SetHeader([]string{"PID", "CMD", "IMAGE", "INSTANCES", "ENDPOINTS", "CPU", "MEM (MB)", "STATUS"})
	} else {
		table.SetHeader([]string{"PID", "INSTANCES", "ENDPOINTS", "STATUS"})

	}
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tw.ALIGN_LEFT)
	table.SetHeaderAlignment(tw.ALIGN_LEFT)
	fmt.Printf("%s\tWorking\n", USER_MSG_INFO)
	go showSpinner(100 * time.Millisecond)
	if myApps != nil && len(myApps) > 0 {
		for _, app := range myApps {
			client := marathonClient(*marathonURL)
			row := []string{}
			appID := app.ID
			appRuntime, err := client.Application(appID)
			if err != nil {
				log.WithFields(log.Fields{"cmd": "ps"}).Debug("Application ", appRuntime.ID, " status not available")
			}
			if !strings.HasPrefix(appID, "/") {
				appID += "/"
			}
			appInstances := strconv.Itoa(*app.Instances)
			appEndpoints := listEndpoints(appRuntime)
			appStatus := marathonAppStatus(client, appRuntime)
			if showAll {
				appCmd := ""
				if app.Cmd != nil {
					appCmd = *app.Cmd
					if len(appCmd) > CMD_TRUNCATE {
						appCmd = appCmd[:CMD_TRUNCATE] + "..."
					}
				} else {
					appCmd = "N/A"
				}
				appImage := ""
				if len(app.Container.Docker.Image) > 0 {
					appImage = app.Container.Docker.Image
				} else {
					appImage = "N/A"
				}
				appCPU := strconv.FormatFloat(app.CPUs, 'f', -1, 64)
				appMem := strconv.FormatFloat(*app.Mem, 'f', -1, 64)
				row = []string{appID, appCmd, appImage, appInstances, appEndpoints, appCPU, appMem, appStatus}
			} else {
				row = []string{appID, appInstances, appEndpoints, appStatus}
			}
			table.Append(row)
		}
		hideSpinner()
		fmt.Printf("%s\tRuntime properties of your app [%s]:\n", USER_MSG_INFO, appDescriptor.AppName)
		table.Render()
	} else {
		fmt.Printf("%s\tDidn't find any processes belonging to your app\n", USER_MSG_PROBLEM)
		return false
	}
	return true
}

// Scale sets the number of instances of a particular µS identified through pid.
func Scale(workdir string, showAll bool, pid string, instances int) bool {
	setLogLevel()
	appDescriptor := readAppDescriptor(workdir)
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"cmd": "scale"}).Error("Failed to connect to Marathon due to error ", err)
		return false
	}
	client := marathonClient(*marathonURL)
	if _, err = client.ScaleApplicationInstances(pid, instances, false); err != nil { // note: not forcing, last parameter set to false
		fmt.Printf("%s\tFailed to scale Marathon app %s due to following error: %s\n", USER_MSG_PROBLEM, pid, err)
		return false
	} else {
		client.WaitOnApplication(pid, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		fmt.Printf("%s\tSuccessfully scaled app %s to %d instances\n", USER_MSG_SUCCESS, pid, instances)
	}
	return true

}
