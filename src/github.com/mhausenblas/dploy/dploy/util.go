package dploy

import (
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	DEFAULT_DEPLOY_WAIT_TIME time.Duration = 10
)

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

func readAppDescriptor() DployApp {
	log.WithFields(nil).Debug("Trying to read app descriptor ", APP_DESCRIPTOR_FILENAME)
	d, err := ioutil.ReadFile(APP_DESCRIPTOR_FILENAME)
	if err != nil {
		log.Fatalf("Failed to read app descriptor. Error: %v", err)
	}
	appDescriptor := DployApp{}
	uerr := yaml.Unmarshal([]byte(d), &appDescriptor)
	if uerr != nil {
		log.Fatalf("Failed to de-serialize app descriptor due to %v", err)
	}
	return appDescriptor
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

func marathonGetInfo(marathonURL url.URL) *marathon.Info {
	client := marathonClient(marathonURL)
	info, err := client.Info()
	if err != nil {
		log.Fatalf("Failed to get Marathon info. Error: %s", err)
	}
	return info
}

func marathonGetApps(marathonURL url.URL) *marathon.Applications {
	client := marathonClient(marathonURL)
	applications, err := client.Applications(url.Values{})
	if err != nil {
		log.Fatalf("Failed to list Marathon apps. Error: %s", err)
	}
	return applications
}

func marathonLaunchApps(marathonURL url.URL) string {
	client := marathonClient(marathonURL)
	applicationName := "helloworld"
	cmd := "env >index.html && python3 -m http.server 8080"
	containerPort := 8080
	resourceRoles := []string{"slave_public"}

	app := marathon.NewDockerApplication()
	app.
		Name(applicationName).
		CPU(0.1).
		Memory(32).
		Count(1).
		Command(cmd).
		Container.Docker.Container("python:3").
		Bridged().
		Expose(containerPort)
	app.AcceptedResourceRoles = resourceRoles

	app, err := client.CreateApplication(app)
	client.WaitOnApplication(app.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)

	if err != nil {
		log.Fatalf("Failed to create application %s. Error: %s", app, err)
	} else {
		log.WithFields(log.Fields{"marathon": "create_app"}).Info("Created app ", app)
	}

	return app.String()
}
