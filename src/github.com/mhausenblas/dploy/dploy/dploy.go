package dploy

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	APP_DESCRIPTOR_FILENAME string = "dploy.app"
	DEFAULT_MARATHON_URL    string = "http://localhost:8080"
	DEFAULT_APP_NAME        string = "CHANGEME"
	MARATHON_APP_SPEC_DIR   string = "specs/"
	TEMPLATE_HELLO_WORLD    string = "https://raw.githubusercontent.com/mhausenblas/dploy/master/templates/helloworld.json"
)

// DployApp is the dploy application deployment descriptor, in short: app descriptor.
// It defines the connection to the target DC/OS cluster as well as the app properties.
type DployApp struct {
	MarathonURL string `yaml:"marathon_url"`
	AppName     string `yaml:"app_name"`
}

func setLogLevel() {
	logLevel := os.Getenv("DPLOY_LOGLEVEL")
	switch strings.ToLower(logLevel) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	default:
		log.SetLevel(log.ErrorLevel)
	}
}

func writeData(fileName string, data string) {
	f, err := os.Create(fileName)
	if err != nil {
		log.WithFields(log.Fields{"template": "download"}).Error("Can't create ", fileName, " due to ", err)
	}
	bytesWritten, err := f.WriteString(data)
	f.Sync()
	log.WithFields(log.Fields{"file": "write"}).Debug("Created ", fileName, ", ", bytesWritten, " Bytes written to disk.")
}

func getTemplate(templateURL url.URL) (string, string) {
	response, err := http.Get(templateURL.String())
	templateFilePath := strings.Split(templateURL.Path, "/")
	templateFileName := templateFilePath[len(templateFilePath)-1]
	if err != nil {
		log.WithFields(log.Fields{"template": "download"}).Error("Can't download template ", templateURL.String(), "due to ", err)
		return templateFileName, ""
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.WithFields(log.Fields{"template": "read"}).Error("Can't read template content due to ", err)
		}
		return templateFileName, string(contents)
	}
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

// DryRun validates the app descriptor by checking if Marathon is reachable.
func DryRun() {
	setLogLevel()
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
	fmt.Printf("🙌\tFound DC/OS Marathon instance\n")
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" name: ", info.Name)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" version: ", info.Version)
	log.WithFields(log.Fields{"cmd": "dryrun"}).Info(" leader: ", info.Leader)
	fmt.Printf("🙌\tFound an app descriptor and an app spec\n")
	fmt.Printf("➡️\tNow you can launch your app using `dploy run`\n")
}
