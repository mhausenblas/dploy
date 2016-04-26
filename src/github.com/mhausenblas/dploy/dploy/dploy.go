package dploy

import (
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/url"
	"os"
)

const (
	APP_DESCRIPTOR_FILENAME string = "dploy.app"
	DEFAULT_MARATHON_URL    string = "http://localhost:8080"
	DEFAULT_APP_NAME        string = "CHANGEME"
)

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

func Init(location string) {
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", location)
	appDescriptor := DployApp{}
	appDescriptor.MarathonURL = DEFAULT_MARATHON_URL
	appDescriptor.AppName = DEFAULT_APP_NAME
	d, err := yaml.Marshal(&appDescriptor)
	if err != nil {
		log.Fatalf("Failed to serialize dploy app descriptor. Error: %v", err)
	}
	log.SetLevel(log.DebugLevel)
	log.WithFields(nil).Debug(APP_DESCRIPTOR_FILENAME, "\n", string(d))
	f, err := os.Create(APP_DESCRIPTOR_FILENAME)
	if err != nil {
		panic(err)
	}
	bytesWritten, err := f.WriteString(string(d))
	f.Sync()
	log.WithFields(log.Fields{"cmd": "init"}).Info("Created ", APP_DESCRIPTOR_FILENAME, ", ", bytesWritten, " Bytes written to disk.")
}

func DryRun() {
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
	log.SetLevel(log.DebugLevel)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info("Found DC/OS Marathon instance")
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" name: ", info.Name)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" version: ", info.Version)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Debug(" leader: ", info.Leader)
}
