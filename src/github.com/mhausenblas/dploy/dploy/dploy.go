package dploy

import (
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const (
	APP_DESCRIPTOR_FILENAME string = "dploy.app"
	DEFAULT_MARATHON_URL    string = "http://localhost:8080"
	DEFAULT_APP_NAME        string = "CHANGEME"
)

// DployApp is the dploy application deployment descriptor, in short: app descriptor.
// It defines the connection to the target DC/OS cluster as well as the app properties.
type DployApp struct {
	MarathonURL string `yaml:"marathon_url"`
	AppName     string `yaml:"app_name"`
}

func marathonClient(marathonURL url.URL) marathon.Marathon {
	config := marathon.NewDefaultConfig()
	config.URL = marathonURL.String()
	client, err := marathon.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create a client for Marathon. Error: %s", err)
	}
	return client
}

func withDebug() {
	doDebug, _ := strconv.ParseBool(os.Getenv("DPLOY_DEBUG"))
	if doDebug {
		log.SetLevel(log.DebugLevel)
	}
}

func marathonGetApps(marathonURL url.URL) *marathon.Applications {
	client := marathonClient(marathonURL)
	applications, err := client.Applications(url.Values{})
	if err != nil {
		log.Fatalf("Failed to list Marathon apps. Error: %s", err)
	}
	return applications
}

func marathonGetInfo(marathonURL url.URL) *marathon.Info {
	client := marathonClient(marathonURL)
	info, err := client.Info()
	if err != nil {
		log.Fatalf("Failed to get Marathon info. Error: %s", err)
	}
	return info
}

// Init creates an app descriptor (dploy.app) in the location specified.
// If no location is provided the app descriptor is created in the current directory.
// If a location is provided, it can be absolute or relative to the dir dploy is executed.
// For example:
//  dploy.Init("../.")
//  dploy.Init("/Users/mhausenblas/")
func Init(location string) {
	withDebug()
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", location)
	appDescriptor := DployApp{}
	appDescriptor.MarathonURL = DEFAULT_MARATHON_URL
	appDescriptor.AppName = DEFAULT_APP_NAME
	d, err := yaml.Marshal(&appDescriptor)
	if err != nil {
		log.Fatalf("Failed to serialize dploy app descriptor. Error: %v", err)
	}
	log.WithFields(nil).Debug("Creating app descriptor ", APP_DESCRIPTOR_FILENAME, " with following content:\n", string(d))

	if location == "" {
		location = "./"
	}
	appDescriptorLocation, _ := filepath.Abs(filepath.Join(location, APP_DESCRIPTOR_FILENAME))
	f, err := os.Create(appDescriptorLocation)
	if err != nil {
		panic(err)
	}
	bytesWritten, err := f.WriteString(string(d))
	f.Sync()
	log.WithFields(log.Fields{"cmd": "init"}).Info("Created ", APP_DESCRIPTOR_FILENAME, ", ", bytesWritten, " Bytes written to disk.")
}

// DryRun validates the app descriptor by checking if Marathon is reachable.
func DryRun() {
	withDebug()
	log.WithFields(nil).Debug("Trying to read app descriptor ", APP_DESCRIPTOR_FILENAME)
	d, err := ioutil.ReadFile(APP_DESCRIPTOR_FILENAME)
	if err != nil {
		log.Fatalf("Failed to read app descriptor. Error: %v", err)
	}
	appDescriptor := DployApp{}
	uerr := yaml.Unmarshal([]byte(d), &appDescriptor)
	if uerr != nil {
		log.Fatalf("error: %v", err)
	}
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.Fatal(err)
	}
	info := marathonGetInfo(*marathonURL)

	log.WithFields(log.Fields{"cmd": "dryrun"}).Info("Found DC/OS Marathon instance")
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" name: ", info.Name)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" version: ", info.Version)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" leader: ", info.Leader)
}
