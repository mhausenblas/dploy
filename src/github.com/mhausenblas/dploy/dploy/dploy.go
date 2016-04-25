package dploy

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	"net/url"
)

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

func Init(location string) {
	log.WithFields(log.Fields{"cmd": "init"}).Info("Init app in dir: ", location)
}

func DryRun() {
	marathonURL, err := url.Parse("http://localhost:8080/v2/apps")
	if err != nil {
		log.Fatal(err)
	}
	applications := marathonGetApps(*marathonURL)
	for _, application := range applications.Apps {
		fmt.Printf("Application: %s", application)
	}
}
