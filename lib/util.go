package dploy

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
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

func ensureWorkDir(workdirPath string) {
	workDir, _ := filepath.Abs(workdirPath)
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		os.MkdirAll(workdirPath, 0755)
	}
}

func writeData(fileName string, data string) {
	f, err := os.Create(fileName)
	if err != nil {
		log.WithFields(log.Fields{"file": "write"}).Error("Can't create ", fileName, " due to ", err)
	}
	bytesWritten, err := f.WriteString(data)
	f.Sync()
	log.WithFields(log.Fields{"file": "write"}).Debug("Created ", fileName, ", ", bytesWritten, " Bytes written to disk.")
}

func getExample(exampleURL url.URL) (string, string) {
	response, err := http.Get(exampleURL.String())
	exampleFilePath := strings.Split(exampleURL.Path, "/")
	exampleFileName := exampleFilePath[len(exampleFilePath)-1]
	if err != nil {
		log.WithFields(log.Fields{"example": "download"}).Error("Can't download example ", exampleURL.String(), "due to ", err)
		return exampleFileName, ""
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.WithFields(log.Fields{"example": "read"}).Error("Can't read example content due to ", err)
		}
		return exampleFileName, string(contents)
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
		log.Fatalf("Failed to de-serialize app descriptor due to %v", uerr)
	}
	return appDescriptor
}

func getAppSpecs(workdir string) []string {
	appSpecDir, _ := filepath.Abs(filepath.Join(workdir, MARATHON_APP_SPEC_DIR))
	log.WithFields(log.Fields{"marathon": "get_app_specs"}).Debug("Trying to find app specs in ", appSpecDir)
	files, _ := ioutil.ReadDir(appSpecDir)
	appSpecs := []string{}
	for _, f := range files {
		fExt := strings.ToLower(filepath.Ext(f.Name()))
		log.WithFields(log.Fields{"marathon": "get_app_specs"}).Debug("Testing ", f.Name(), " with extension ", fExt)
		if !f.IsDir() && (strings.Compare(fExt, MARATHON_APP_SPEC_EXT) == 0) {
			log.WithFields(log.Fields{"marathon": "get_app_specs"}).Debug("Found app spec ", f.Name())
			appSpecFilename, _ := filepath.Abs(filepath.Join(appSpecDir, f.Name()))
			appSpecs = append(appSpecs, appSpecFilename)
			log.WithFields(log.Fields{"marathon": "get_app_specs"}).Debug("Added app spec: ", appSpecFilename)
		}
	}
	return appSpecs
}

func readAppSpec(appSpecFilename string) *marathon.Application {
	log.WithFields(log.Fields{"marathon": "read_app_spec"}).Debug("Trying to read app spec ", appSpecFilename)
	d, err := ioutil.ReadFile(appSpecFilename)
	if err != nil {
		log.WithFields(log.Fields{"marathon": "read_app_spec"}).Error("Can't read app spec ", appSpecFilename)
	}
	log.WithFields(log.Fields{"marathon": "read_app_spec"}).Debug("Got app spec:\n", string(d))
	app := marathon.Application{}
	uerr := json.Unmarshal([]byte(d), &app)
	if uerr != nil {
		log.Fatalf("Failed to de-serialize app spec due to %v", uerr)
	}
	return &app
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

func marathonAppStatus(marathonURL url.URL, appID string) string {
	client := marathonClient(marathonURL)
	details, err := client.Application(appID)
	if err != nil {
		log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appID, " not running")
		return SYSTEM_MSG_OFFLINE
	} else {
		if details.Tasks != nil && len(details.Tasks) > 0 {
			health, _ := client.ApplicationOK(details.ID)
			log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", details.ID, "health status: ", health)
			if health {
				return SYSTEM_MSG_ONLINE
			} else {
				return SYSTEM_MSG_OFFLINE
			}
		} else {
			return SYSTEM_MSG_OFFLINE
		}
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

func marathonCreateApps(marathonURL url.URL, workdir string) {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec := readAppSpec(specFilename)
		app, err := client.CreateApplication(appSpec)
		if err != nil {
			log.WithFields(log.Fields{"marathon": "create_app"}).Error("Failed to create application due to:\n\t", err)
			log.Fatalf("Exiting for now; try running `dploy destroy` first.")
		} else {
			log.WithFields(log.Fields{"marathon": "create_app"}).Info("Created app ", app.ID)
			log.WithFields(log.Fields{"marathon": "create_app"}).Debug("App deployment: ", app)
		}
		client.WaitOnApplication(app.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
	}
}

func marathonDeleteApps(marathonURL url.URL, workdir string) {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec := readAppSpec(specFilename)
		_, err := client.DeleteApplication(appSpec.ID)
		if err != nil {
			log.Fatalf("Failed to create application %s. Error: %s", appSpec.ID, err)
		} else {
			log.WithFields(log.Fields{"marathon": "create_app"}).Info("Deleted app ", appSpec.ID)
		}
		client.WaitOnDeployment(appSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
	}
}
