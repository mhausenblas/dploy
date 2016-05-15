package dploy

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
	tw "github.com/olekukonko/tablewriter"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Download(theURL string, intoDir string) (string, error) {
	u, _ := url.Parse(theURL)
	fn, c, err := fetch(*u)
	if err != nil {
		return "", err
	}
	writeData(filepath.Join(intoDir, fn), c)
	return fn, nil
}

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

func showSpinner(delay time.Duration) {
	// cursor manipulation see http://shiroyasha.io/escape-sequences-a-quick-guide.html
	fmt.Printf("\033[1A")
	fmt.Printf("\033[2C")
	for {
		for _, r := range `-\|/` {
			fmt.Printf("\r%c", r)
			time.Sleep(delay)
		}
	}
}

func hideSpinner() {
	fmt.Printf("\033[2D")
}

func ensureWorkDir(workdirPath string) {
	workDir, _ := filepath.Abs(workdirPath)
	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		os.MkdirAll(workdirPath, 0755)
	}
	log.WithFields(log.Fields{"workspace": "check"}).Info("Made sure ", workDir, " exists ")
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

func fetch(theURL url.URL) (string, string, error) {
	response, err := http.Get(theURL.String())
	fp := strings.Split(theURL.Path, "/")
	fn := fp[len(fp)-1]
	if err != nil {
		log.WithFields(log.Fields{"fetch": "GET"}).Error("Can't fetch content from due to ", err)
		return fn, "", fmt.Errorf("Can't fetch content from due to %s", err)
	} else {
		defer response.Body.Close()
		c, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.WithFields(log.Fields{"fetch": "read"}).Error("Can't read content due to ", err)
			return fn, "", fmt.Errorf("Can't read content from due to %s", err)
		}
		return fn, string(c), nil
	}
}

func readAppDescriptor(workdir string) DployApp {
	ad, _ := filepath.Abs(filepath.Join(workdir, APP_DESCRIPTOR_FILENAME))
	log.WithFields(log.Fields{"appdescriptor": "read"}).Debug("Trying to read app descriptor ", ad)
	d, err := ioutil.ReadFile(ad)
	if err != nil {
		log.Fatalf("Failed to read app descriptor due to %v", err)
	}
	appDescriptor := DployApp{}
	uerr := yaml.Unmarshal([]byte(d), &appDescriptor)
	if uerr != nil {
		log.Fatalf("Failed to de-serialize app descriptor due to %v", uerr)
	}
	log.WithFields(log.Fields{"appdescriptor": "read"}).Debug("Got valid app descriptor ")
	return appDescriptor
}

func getPAT(workdir string) (string, bool) {
	pat, _ := filepath.Abs(filepath.Join(workdir, MARATHON_OBSERVER_PAT_FILE))
	patoken, err := ioutil.ReadFile(pat)
	if err != nil {
		log.WithFields(log.Fields{"pat": "read"}).Debug("Can't read GitHub Personal Access Token file ", err)
		return "", false
	}
	return string(patoken), true
}

func observerAlive(marathonURL url.URL, appID string) bool {
	client := marathonClient(marathonURL)
	appRuntime, err := client.Application(appID)
	if err != nil {
		log.WithFields(log.Fields{"observer": "check"}).Debug("Observer status not available")
		return false
	}
	if appRuntime.Tasks != nil && len(appRuntime.Tasks) > 0 {
		health, _ := client.ApplicationOK(appRuntime.ID)
		if health {
			log.WithFields(log.Fields{"observer": "check"}).Debug("Observer is healthy")
			return true
		} else {
			log.WithFields(log.Fields{"observer": "check"}).Debug("Observer is NOT healthy")
			return false
		}
	} else {
		log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Observer NO TASKS found")
		return false
	}
}

func launchObserver(appDescriptor DployApp, workdir string) bool {
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"observer": "launch"}).Error("Failed to connect to Marathon due to ", err)
		return false
	}
	client := marathonClient(*marathonURL)
	patoken, patExists := getPAT(workdir)
	if appDescriptor.RepoURL != "" && appDescriptor.PublicNode != "" && patExists {
		// https://github.com/OWNER/REPO -> OWNER, REPO
		owpo := appDescriptor.RepoURL[len("https://github.com/"):]
		owner := strings.Split(owpo, "/")[0]
		repo := strings.Split(owpo, "/")[1]
		log.WithFields(log.Fields{"observer": "launch"}).Debug("Got repo ", owner, "/", repo)
		fn, err := Download(MARATHON_OBSERVER_TEMPLATE, workdir)
		if err != nil {
			log.WithFields(log.Fields{"observer": "launch"}).Error("Failed to download observer template due to ", err)
			return false
		}
		observerTemplate, _ := filepath.Abs(filepath.Join(workdir, fn))
		appSpec, _ := readAppSpec(appDescriptor.AppName, observerTemplate)

		if ok := observerAlive(*marathonURL, appSpec.ID); ok {
			return true
		} else {
			appSpec.AddEnv("DPLOY_PUBLIC_NODE", appDescriptor.PublicNode)
			appSpec.AddEnv("DPLOY_OBSERVER_GITHUB_PAT", patoken)
			appSpec.AddEnv("DPLOY_OBSERVER_GITHUB_OWNER", owner)
			appSpec.AddEnv("DPLOY_OBSERVER_GITHUB_REPO", repo)
			if branch := appDescriptor.TriggerBranch; branch != "" {
				appSpec.AddEnv("DPLOY_OBSERVER_TARGETBRANCH", branch)
			}
			log.WithFields(log.Fields{"observer": "launch"}).Debug("Trying to launch observer with following app spec ", appSpec)
			if _, err := os.Stat(observerTemplate); err == nil {
				os.Remove(observerTemplate)
				log.WithFields(log.Fields{"observer": "launch"}).Debug("Removed temporary observer template ", observerTemplate)
			}
			if appSpec != nil {
				app, err := client.CreateApplication(appSpec)
				if err != nil {
					log.WithFields(log.Fields{"observer": "launch"}).Error("Failed to launch observer due to ", err)
					return false
				} else {
					log.WithFields(log.Fields{"observer": "launch"}).Info("Launched observer with ID ", app.ID)
				}
				client.WaitOnApplication(app.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
				if err != nil {
					log.WithFields(log.Fields{"observer": "launch"}).Error("Failed to connect to Marathon due to ", err)
					return false
				}
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func killObserver(appDescriptor DployApp, workdir string) bool {
	marathonURL, err := url.Parse(appDescriptor.MarathonURL)
	if err != nil {
		log.WithFields(log.Fields{"observer": "kilil"}).Error("Failed to connect to Marathon due to ", err)
		return false
	}
	client := marathonClient(*marathonURL)
	if appDescriptor.RepoURL != "" && appDescriptor.PublicNode != "" {
		fn, err := Download(MARATHON_OBSERVER_TEMPLATE, workdir)
		if err != nil {
			log.WithFields(log.Fields{"observer": "kill"}).Error("Failed to download observer template due to ", err)
			return false
		}
		observerTemplate, _ := filepath.Abs(filepath.Join(workdir, fn))
		appSpec, _ := readAppSpec(appDescriptor.AppName, observerTemplate)
		if ok := observerAlive(*marathonURL, appSpec.ID); ok {
			_, err := client.DeleteApplication(appSpec.ID)
			if err != nil {
				log.WithFields(log.Fields{"observer": "kill"}).Info("Failed to kill observer")
				return false
			}
			log.WithFields(log.Fields{"observer": "kill"}).Info("Killed observer")
			client.WaitOnDeployment(appSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
			// TODO: unregister Webhook as well
			return true
		}
	}
	return false
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
	log.WithFields(log.Fields{"marathon": "label_app"}).Debug("Owning app ", app.ID)
	app.AddLabel(MARATHON_LABEL, label)
}

func renderAppResources(appDescriptor DployApp, workdir string) {
	table := tw.NewWriter(os.Stdout)
	row := []string{"Marathon", RESOURCETYPE_PLATFORM, appDescriptor.MarathonURL}
	table.Append(row)
	if appSpecs := getAppSpecs(workdir); len(appSpecs) > 0 {
		for _, specFilename := range appSpecs {
			appSpec, groupAppSpec := readAppSpec(appDescriptor.AppName, specFilename)
			if appSpec != nil { // we have an app
				renderApp(appSpec, specFilename, "", table)
			} else { // we have a group
				renderGroup(groupAppSpec, specFilename, "", table)
			}
		}
		fmt.Printf("%s\tResources of your app [%s]:\n", USER_MSG_INFO, appDescriptor.AppName)
		table.SetHeader([]string{"RESOURCE", "TYPE", "LOCATION"})
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
}

func renderApp(app *marathon.Application, specFilename string, path string, table *tw.Table) {
	appID := app.ID
	if !strings.HasPrefix(app.ID, "/") {
		appID = path + "/" + app.ID
	}
	log.WithFields(log.Fields{"render": "app"}).Debug("In app ", app.ID)
	resType := RESOURCETYPE_APP
	row := []string{appID, resType, "./" + MARATHON_APP_SPEC_DIR + strings.Split(specFilename, MARATHON_APP_SPEC_DIR)[1]}
	table.Append(row)
}

func renderGroup(group *marathon.Group, specFilename string, path string, table *tw.Table) {
	resType := RESOURCETYPE_GROUP
	groupID := group.ID
	if !strings.HasPrefix(group.ID, "/") {
		groupID = path + "/" + group.ID
	}
	path = groupID
	log.WithFields(log.Fields{"render": "group"}).Debug("At node ", path)
	row := []string{groupID, resType, "./" + MARATHON_APP_SPEC_DIR + strings.Split(specFilename, MARATHON_APP_SPEC_DIR)[1]}
	table.Append(row)
	// process the rest of the members of this branch:
	if group.Apps != nil {
		for _, app := range group.Apps {
			renderApp(app, specFilename, path, table)
		}
	}
	if group.Groups != nil {
		for _, g := range group.Groups {
			renderGroup(g, specFilename, path, table)
		}
	}
}

func listEndpoints(app *marathon.Application) string {
	var endpoints []string
	for _, task := range app.Tasks {
		log.WithFields(log.Fields{"endpoints": "list"}).Debug("Inspecting task ", task)
		endpoints = append(endpoints, fmt.Sprintf("%s:%d", task.Host, task.Ports[0]))
	}
	return strings.Join(endpoints[:], " ")
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

func marathonAppStatus(client marathon.Marathon, appRuntime *marathon.Application) string {
	log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appRuntime)
	if appRuntime.Tasks != nil && len(appRuntime.Tasks) > 0 {
		health, _ := client.ApplicationOK(appRuntime.ID)
		if health {
			log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appRuntime.ID, " is healthy")
			return SYSTEM_MSG_ONLINE
		} else {
			log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appRuntime.ID, " is NOT healthy")
			return SYSTEM_MSG_OFFLINE
		}
	} else {
		log.WithFields(log.Fields{"marathon": "app_status"}).Debug("Application ", appRuntime.ID, " NO TASKS found")
		return SYSTEM_MSG_OFFLINE
	}
}

func marathonAppRuntime(marathonURL url.URL, dployAppName string) []marathon.Application {
	client := marathonClient(marathonURL)
	applications, err := client.Applications(nil)
	var myApps []marathon.Application
	if err != nil {
		log.Fatalf("Failed to list Marathon apps. Error: %s", err)
	}
	for _, app := range applications.Apps {
		if app.Labels != nil {
			log.WithFields(log.Fields{"marathon": "app_runtime"}).Debug("Checking app ", app.ID, " against label ", dployAppName)
			for k, v := range *app.Labels {
				log.WithFields(log.Fields{"marathon": "app_runtime"}).Debug("LABEL: ", k, " VALUE: ", v)
				if k == MARATHON_LABEL && v == dployAppName {
					myApps = append(myApps, app)
				}
			}
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
				log.WithFields(log.Fields{"marathon": "create_app"}).Error("Failed to create app due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("Created app ", app.ID)
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("App deployment: ", app)
			}
			client.WaitOnApplication(app.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		} else {
			err := client.CreateGroup(group)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "create_app"}).Error("Failed to create group due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("Created group ", group.ID)
				log.WithFields(log.Fields{"marathon": "create_app"}).Debug("App deployment: ", group)
			}
			client.WaitOnGroup(group.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		}
	}
}

func marathonUpdateApps(marathonURL url.URL, dployAppName string, workdir string) error {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec, _ := readAppSpec(dployAppName, specFilename)
		log.WithFields(log.Fields{"marathon": "update_app"}).Debug("Looking at ", dployAppName, " in ", specFilename)
		if appSpec != nil {
			//TODO: only update apps that have actually changed
			_, err := client.UpdateApplication(appSpec, true) // note: for now we default to force updates
			if err != nil {
				log.WithFields(log.Fields{"marathon": "update_app"}).Error("Failed to update app due to ", err)
				return err
			} else {
				log.WithFields(log.Fields{"marathon": "update_app"}).Debug("Updated app: ", appSpec.ID)
			}
			client.WaitOnApplication(appSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		}
	}
	return nil
}

func marathonDeleteApps(marathonURL url.URL, dployAppName string, workdir string) {
	client := marathonClient(marathonURL)
	appSpecs := getAppSpecs(workdir)
	for _, specFilename := range appSpecs {
		appSpec, groupAppSpec := readAppSpec(dployAppName, specFilename)
		if appSpec != nil {
			_, err := client.DeleteApplication(appSpec.ID)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Failed to delete app ", appSpec.ID, " due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Deleted app ", appSpec.ID)
			}
			client.WaitOnDeployment(appSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		} else {
			_, err := client.DeleteGroup(groupAppSpec.ID)
			if err != nil {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Failed to delete group ", groupAppSpec.ID, " due to ", err)
			} else {
				log.WithFields(log.Fields{"marathon": "delete_app"}).Info("Deleted group ", groupAppSpec.ID)
			}
			client.WaitOnDeployment(groupAppSpec.ID, DEFAULT_DEPLOY_WAIT_TIME*time.Second)
		}
	}
}
