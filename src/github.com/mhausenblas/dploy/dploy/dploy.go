package dploy

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path/filepath"
)

const (
	APP_DESCRIPTOR_FILENAME string = "dploy.app"
	DEFAULT_MARATHON_URL    string = "http://localhost:8080"
	DEFAULT_APP_NAME        string = "CHANGEME"
	MARATHON_APP_SPEC_DIR   string = "specs/"
	MARATHON_APP_SPEC_EXT   string = "json"
	TEMPLATE_HELLO_WORLD    string = "https://raw.githubusercontent.com/mhausenblas/dploy/master/templates/helloworld.json"
)

// DployApp is the dploy application deployment descriptor, in short: app descriptor.
// It defines the connection to the target DC/OS cluster as well as the app properties.
type DployApp struct {
	MarathonURL string `yaml:"marathon_url"`
	AppName     string `yaml:"app_name"`
}

// Init creates an app descriptor (dploy.app) in the location specified.
// If no location is provided the app descriptor is created in the current directory.
// If a location is provided, it can be absolute or relative to the dir dploy is executed.
// For example:
//  dploy.Init("../.")
//  dploy.Init("/Users/mhausenblas/")
func Init(location string) {
	setLogLevel()
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", location)
	appDescriptor := DployApp{}
	appDescriptor.MarathonURL = DEFAULT_MARATHON_URL
	appDescriptor.AppName = DEFAULT_APP_NAME
	d, err := yaml.Marshal(&appDescriptor)
	if err != nil {
		log.Fatalf("Failed to serialize dploy app descriptor. Error: %v", err)
	}
	log.WithFields(nil).Debug("Trying to create app descriptor ", APP_DESCRIPTOR_FILENAME, " with following content:\n", string(d))
	if location == "" {
		location = "./"
	}
	appDescriptorLocation, _ := filepath.Abs(filepath.Join(location, APP_DESCRIPTOR_FILENAME))
	writeData(appDescriptorLocation, string(d))
	specsDir, _ := filepath.Abs(filepath.Join(location, MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		os.Mkdir(specsDir, 0755)
		log.WithFields(log.Fields{"cmd": "init"}).Info("Created ", specsDir)
	}
	templateURL, err := url.Parse(TEMPLATE_HELLO_WORLD)
	templateFileName, templateContent := getTemplate(*templateURL)
	writeData(filepath.Join(specsDir, templateFileName), templateContent)
	fmt.Printf("🙌\tDone initializing your app:\n")
	fmt.Printf(" I set up app descriptor in %s\n", appDescriptorLocation)
	fmt.Printf(" I created app spec directory %s\n", specsDir)
	fmt.Printf(" I initialized app spec directory with %s\n", templateFileName)
	fmt.Printf("➡️\tNow it's time to edit the app descriptor and adapt or add Marathon app specs. Next, you can run `dploy dryrun`\n")
}

// DryRun validates the app descriptor by checking if Marathon is reachable and also
// checks if the app spec directory is present, incl. at least one Marathon app spec.
func DryRun() {
	setLogLevel()
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	info := marathonGetInfo(*marathonURL)
	fmt.Printf("🙌\tFound DC/OS Marathon instance\n")
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" name: ", info.Name)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" version: ", info.Version)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" leader: ", info.Leader)

	specsDir, _ := filepath.Abs(filepath.Join("./", MARATHON_APP_SPEC_DIR))
	if _, err := os.Stat(specsDir); os.IsNotExist(err) {
		fmt.Printf("🙁\tDidn't find app spec dir, expecting it in %s\n", specsDir)
		fmt.Printf("➡️\tDid you do `dploy init` here?\n")
		os.Exit(3)
	} else {
		appSpecs := getAppSpecs()
		for _, specFilename := range appSpecs {
			appSpec := readAppSpec(specFilename)
			if err != nil {
				log.Fatalf("Failed to create application %s. Error: %s", appSpec, err)
			} else {
				log.WithFields(log.Fields{"marathon": "read_app"}).Info("Found app spec ", appSpec)
			}
		}
		fmt.Printf("🙌\tFound an app descriptor and app spec(s)\n")
	}

	fmt.Printf("➡️\tNow you can launch your app using `dploy run`\n")
}

// Run launches the app using the Marathon API
func Run() {
	setLogLevel()
	appDescriptor := readAppDescriptor()
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	apps := marathonLaunchApps(*marathonURL)
	fmt.Printf("🙌\tLaunched %s\n", apps)
}
