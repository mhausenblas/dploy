package dploy

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	tw "github.com/olekukonko/tablewriter"
	yaml "gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ENV_VAR_DPLOY_LOGLEVEL   string        = "DPLOY_LOGLEVEL"
	ENV_VAR_DPLOY_EXAMPLES   string        = "DPLOY_EXAMPLES"
	DEFAULT_DEPLOY_WAIT_TIME time.Duration = 10
	APP_DESCRIPTOR_FILENAME  string        = "dploy.app"
	DEFAULT_MARATHON_URL     string        = "http://localhost:8080"
	DEFAULT_APP_NAME         string        = "CHANGEME"
	MARATHON_APP_SPEC_DIR    string        = "specs/"
	MARATHON_APP_SPEC_EXT    string        = ".json"
	EXAMPLE_HELLO_WORLD      string        = "https://raw.githubusercontent.com/mhausenblas/dploy/master/examples/helloworld.json"
	EXAMPLE_BUZZ             string        = "https://raw.githubusercontent.com/mhausenblas/dploy/master/examples/buzz/buzz.json"
	USER_MSG_SUCCESS         string        = "🙌"
	USER_MSG_PROBLEM         string        = "🙁"
	USER_MSG_INFO            string        = "🗣"
	SYSTEM_MSG_ONLINE        string        = "online\t💚"
	SYSTEM_MSG_OFFLINE       string        = "offline\t💔"
)

// DployApp is the dploy application deployment descriptor, in short: app descriptor.
// It defines the connection to the target DC/OS cluster as well as the app properties.
type DployApp struct {
	MarathonURL string `yaml:"marathon_url"`
	AppName     string `yaml:"app_name"`
}

// Init creates an app descriptor (dploy.app) and the `specs/` directory
// in the workdir specified as well as copies in example app specs.
// For example:
//  dploy.Init("../.")
func Init(workdir string) {
	setLogLevel()
	ensureWorkDir(workdir)
	fmt.Printf("%s\tInitializing your app ...\n", USER_MSG_INFO)
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", workdir)
	appDescriptor := DployApp{}
	appDescriptor.MarathonURL = DEFAULT_MARATHON_URL
	appDescriptor.AppName = DEFAULT_APP_NAME
	d, err := yaml.Marshal(&appDescriptor)
	if err != nil {
		log.Fatalf("Failed to serialize dploy app descriptor. Error: %v", err)
	}
	log.WithFields(nil).Debug("Trying to create app descriptor ", APP_DESCRIPTOR_FILENAME, " with following content:\n", string(d))
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
		fetchExample(EXAMPLE_HELLO_WORLD, specsDir)
		fetchExample(EXAMPLE_BUZZ, specsDir)
		fmt.Printf("\t\tInitialized app spec directory with some examples\n")
	case "buzz":
		fetchExample(EXAMPLE_BUZZ, specsDir)
		fmt.Printf("\t\tInitialized app spec directory with the buzz example\n")
	default:
	}
	fmt.Printf("%s\tNow it's time to edit the app descriptor and adapt or add Marathon app specs. Next, you can run `dploy dryrun`\n", USER_MSG_INFO)
}

// DryRun validates the app descriptor by checking if Marathon is reachable and also
// checks if the app spec directory is present, incl. at least one Marathon app spec.
func DryRun(workdir string) {
	setLogLevel()
	ensureWorkDir(workdir)
	fmt.Printf("%s\tKicking the tires! Checking Marathon connection, descriptor and app specs ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
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
		os.Exit(3)
	} else {
		appDescriptor := readAppDescriptor()
		if strings.HasPrefix(appDescriptor.MarathonURL, "http") {
			fmt.Printf("%s\tFound an app descriptor\n", USER_MSG_SUCCESS)
			if appSpecs := getAppSpecs(workdir); len(appSpecs) > 0 {
				fmt.Printf("%s\tFound %d app spec(s) to deploy\n", USER_MSG_SUCCESS, len(appSpecs))
			} else {
				fmt.Printf("%s\tDidn't find any app specs in %s \n", USER_MSG_PROBLEM, MARATHON_APP_SPEC_DIR)
				os.Exit(3)
			}
		} else {
			fmt.Printf("%s\tDidn't find an app descriptor (%s) in current directory\n", USER_MSG_PROBLEM, APP_DESCRIPTOR_FILENAME)
			os.Exit(3)
		}
	}
	fmt.Printf("%s\tNow you can launch your app using `dploy run`\n", USER_MSG_INFO)
}

// Run launches the app as defined in the descriptor and the app specs.
// It scans the `specs/` directory for Marathon app specs and launches them using the Marathon API.
func Run(workdir string) {
	setLogLevel()
	ensureWorkDir(workdir)
	fmt.Printf("%s\tOK, let's rock and roll! Trying to launch your app ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	marathonCreateApps(*marathonURL, workdir)
	fmt.Printf("%s\tLaunched your app!\n", USER_MSG_SUCCESS)
	fmt.Printf("%s\tNow you can use `dploy ls` to list resources or `dploy destroy` to tear down the app again.\n", USER_MSG_INFO)
}

// Destroy tears down the app.
// It scans the `specs/` directory for Marathon app specs and deletes apps using the Marathon API.
func Destroy(workdir string) {
	setLogLevel()
	ensureWorkDir(workdir)
	fmt.Printf("%s\tSeems you wanna get rid of your app. OK, gonna try and tear it down now ...\n", USER_MSG_INFO)
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	marathonDeleteApps(*marathonURL, workdir)
	fmt.Printf("%s\tDestroyed your app!\n", USER_MSG_SUCCESS)
}

// ListResources lists the resource definitions of the app.
func ListResources(workdir string) {
	setLogLevel()
	ensureWorkDir(workdir)
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	specsDir, _ := filepath.Abs(filepath.Join(workdir, MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		fmt.Printf("%s\tDidn't find app spec dir, expecting it in %s\n", USER_MSG_PROBLEM, specsDir)
		fmt.Printf("%s\tTry `dploy init` here first.\n", USER_MSG_INFO)
		os.Exit(3)
	} else {
		appDescriptor := readAppDescriptor()
		if strings.HasPrefix(appDescriptor.MarathonURL, "http") {
			table := tw.NewWriter(os.Stdout)
			row := []string{"Marathon", "PLATFORM", marathonURL.String(), SYSTEM_MSG_ONLINE}
			table.Append(row)
			if appSpecs := getAppSpecs(workdir); len(appSpecs) > 0 {
				for _, specFilename := range appSpecs {
					appSpec, groupAppSpec := readAppSpec(specFilename)
					appID := ""
					resType := ""
					appStatus := ""
					if appSpec != nil {
						appStatus = marathonAppStatus(*marathonURL, appSpec.ID, false)
						resType = "APP"
						appID = appSpec.ID
						if !strings.HasPrefix(appSpec.ID, "/") {
							appID = "/" + appSpec.ID
						}
					} else {
						appStatus = marathonAppStatus(*marathonURL, groupAppSpec.ID, true)
						resType = "GROUP"
						appID = groupAppSpec.ID
						if !strings.HasPrefix(groupAppSpec.ID, "/") {
							appID = "/" + groupAppSpec.ID
						}
					}
					row := []string{appID, resType, MARATHON_APP_SPEC_DIR + strings.Split(specFilename, MARATHON_APP_SPEC_DIR)[1], appStatus}
					table.Append(row)
				}
				fmt.Printf("%s\tResources of your app %s ...\n", USER_MSG_INFO, appDescriptor.AppName)
				table.SetHeader([]string{"RESOURCE", "TYPE", "LOCATION", "STATUS"})
				table.SetCenterSeparator("")
				table.SetColumnSeparator("")
				table.SetRowSeparator("")
				table.SetAlignment(tw.ALIGN_LEFT)
				table.SetHeaderAlignment(tw.ALIGN_LEFT)
				table.Render()
			} else {
				fmt.Printf("%s\tDidn't find any app specs in %s \n", USER_MSG_PROBLEM, MARATHON_APP_SPEC_DIR)
				os.Exit(3)
			}
		} else {
			fmt.Printf("%s\tDidn't find an app descriptor (%s) in current directory\n", USER_MSG_PROBLEM, APP_DESCRIPTOR_FILENAME)
			os.Exit(3)
		}
	}
}
