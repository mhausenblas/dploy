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
	logLevel := os.Getenv(ENV_VAR_DPLOY_LOGLEVEL)
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

func fetchExample(example string, specsDir string) {
	exampleURL, _ := url.Parse(example)
	exampleFileName, exampleContent := getExample(*exampleURL)
	writeData(filepath.Join(specsDir, exampleFileName), exampleContent)
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

func readAppSpec(dployAppName, appSpecFilename string) (*marathon.Application, *marathon.Group) {
	log.WithFields(log.Fields{"marathon": "read_app_spec"}).Debug("Trying to read app spec ", appSpecFilename)
	d, err := ioutil.ReadFile(appSpecFilename)
	if err != nil {
		log.WithFields(log.Fields{"marathon": "read_app_spec"}).Error("Can't read app spec ", appSpecFilename)
	}
	log.WithFields(log.Fields{"marathon": "read_app_spec"}).Debug("Got app spec:\n", string(d))

	if strings.Contains(string(d), "groups") { // we're dealing with a group ; this sniffing is a horrible hack
		group := marathon.Group{}
		uerr := json.Unmarshal([]byte(d), &group)
		if uerr != nil {
			log.Fatalf("Failed to de-serialize app spec for group due to %v", uerr)
		}
		labelGroup(&group, dployAppName)
		return nil, &group
	} else { // we're dealing with a simple app
		app := marathon.Application{}
		uerr := json.Unmarshal([]byte(d), &app)
		if uerr != nil {
			log.Fatalf("Failed to de-serialize app spec due to %v", uerr)
		}
		log.WithFields(log.Fields{"marathon": "read_app_spec"}).Debug("Owning app ", app.ID)
		labelApp(&app, dployAppName)
		return &app, nil
	}
}

func labelGroup(group *marathon.Group, label string) {
	log.WithFields(log.Fields{"marathon": "label_group"}).Debug("In group ", group.ID)
	if group.Apps != nil {
		for _, app := range group.Apps {
			labelApp(app, label)
		}
	}
	if group.Groups != nil {
		for _, g := range group.Groups {
			labelGroup(g, label)
		}
	}
}

func labelApp(app *marathon.Application, label string) {
	log.WithFields(log.Fields{"marathon": "label_group"}).Debug("Owning app ", app.ID)
	app.AddLabel(MARATHON_LABEL, label)
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

func marathonAppStatus(marathonURL url.URL, appID string, isGroup bool) string {
	client := marathonClient(marathonURL)

	if isGroup {
		gExists, _ := client.HasGroup(appID)
		if !gExists {
			log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Group ", appID, " does not exist")
			return SYSTEM_MSG_OFFLINE
		} else {
			_, err := client.Group(appID)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Group ", appID, " not running")
				return SYSTEM_MSG_OFFLINE
			} else {
				return SYSTEM_MSG_ONLINE
			}
		}
	} else {
		details, err := client.Application(appID)
		if err != nil {
			log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appID, " not running")
			return SYSTEM_MSG_OFFLINE
		} else {
			if details.Tasks != nil && len(details.Tasks) > 0 {
				health, _ := client.ApplicationOK(details.ID)
				log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", details.ID, " health status: ", health)
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
}

func marathonAppRuntime(marathonURL url.URL, dployAppName string) []string {
	client := marathonClient(marathonURL)
	applications, err := client.Applications(nil)
	var myApps []string
	if err != nil {
		log.Fatalf("Failed to list Marathon apps. Error: %s", err)
	}
	for _, app := range applications.Apps {
		log.WithFields(log.Fields{"marathon": "app_runtime"}).Debug("Checking application ", app.ID, " against label ", dployAppName)
		if app.Labels != nil {
			myApps = append(myApps, app.ID)
		}
	}
	return myApps
}

func marathonCreateApps(marathonURL url.URL, dployAppName string, workdir string) {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec, group := readAppSpec(dployAppName, specFilename)
		if appSpec != nil {
			app, err := client.CreateApplication(appSpec)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "create_app"}).Error("Failed to create application due to:\n\t", err)
				log.Fatalf("Exiting for now; try running `dploy destroy` first.")
			} else {
				log.WithFields(log.Fields{"marathon": "create_app"}).Info("Created app ", app.ID)
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("App deployment: ", app)
			}
			client.WaitOnApplication(app.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		} else {
			err := client.CreateGroup(group)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "create_app"}).Error("Failed to create group due to:\n\t", err)
				log.Fatalf("Exiting for now; try running `dploy destroy` first.")
			} else {
				log.WithFields(log.Fields{"marathon": "create_app"}).Info("Created group ", group.ID)
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("App deployment: ", group)
			}
			client.WaitOnGroup(group.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		}
	}
}

func marathonDeleteApps(marathonURL url.URL, dployAppName string, workdir string) {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec, groupAppSpec := readAppSpec(dployAppName, specFilename)
		if appSpec != nil {
			_, err := client.DeleteApplication(appSpec.ID)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Error("Failed to delete application ", appSpec.ID, " due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Deleted app ", appSpec.ID)
			}
			client.WaitOnDeployment(appSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		} else {
			_, err := client.DeleteGroup(groupAppSpec.ID)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Error("Failed to delete group ", groupAppSpec.ID, " due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Deleted group ", groupAppSpec.ID)
			}
			client.WaitOnDeployment(groupAppSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		}
	}
}
