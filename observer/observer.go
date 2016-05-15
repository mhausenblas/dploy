package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	github "github.com/google/go-github/github"
	dploy "github.com/mhausenblas/dploy/lib"
	"golang.org/x/oauth2"
	yaml "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	VERSION string = "1.0.2"
	// which branch to observe for changes:
	DEFAULT_OBSERVE_BRANCH string = "dcos"
	// how long to wait (in sec) after launch to register Webhook:
	DEFAULT_OBSERVER_WAIT_TIME time.Duration = 30
)

var (
	// the HTTP request multiplexer
	mux *http.ServeMux

	// the GitHub client
	client *github.Client

	// personal access token (pat), owner and repo to observe
	pat, owner, repo string

	// public IP address FQDN of the public agent this service using
	pubnode string

	// Web hook used to trigger deployment
	deployHook *github.Hook

	// how long to wait to register Webhook (default: DEFAULT_OBSERVER_WAIT_TIME)
	registerDelay time.Duration

	// which branch to observe for push events (default: DEFAULT_OBSERVE_BRANCH)
	targetBranch string

	// time stamp of the last successful deployment
	lastDeployment time.Time
)

type Status struct {
	Owner        string    `json:"owner"`
	Repo         string    `json:"repo"`
	TargetBranch string    `json:"branch"`
	Pubnode      string    `json:"pubnode"`
	LastDeploy   time.Time `json:"lastdeploy"`
}

type DployResult struct {
	Success bool   `json:"success"`
	Msg     string `json:"message"`
}

type DNSResults struct {
	SRVRecords []SRVRecord
}

type SRVRecord struct {
	Service string
	Host    string
	IP      string
	Port    string
}

type HostRecord struct {
	Host string
	IP   string
}

func init() {
	mux = http.NewServeMux()
	registerDelay = DEFAULT_OBSERVER_WAIT_TIME
	targetBranch = DEFAULT_OBSERVE_BRANCH
	grabEnv() // try via env variables first
	flag.StringVar(&pat, "pat", pat, "the personal access token, via https://github.com/settings/tokens")
	flag.StringVar(&owner, "owner", owner, "the GitHub owner, for example 'mhausenblas' or 'mesosphere'.")
	flag.StringVar(&repo, "repo", repo, "the GitHub repo, for example 'dploy' or 'marathon'.")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	cwd, _ := os.Getwd()
	err := stashPAT(cwd)
	if err != nil {
		log.WithFields(log.Fields{"observer": "init"}).Error("GitHub Personal Access Token file not available due to ", err)
	} else {
		lastDeployment = time.Now()
	}
}

func stashPAT(workdir string) error {
	patFile, _ := filepath.Abs(filepath.Join(workdir, dploy.MARATHON_OBSERVER_PAT_FILE))
	f, err := os.Create(patFile)
	if err != nil {
		log.WithFields(log.Fields{"pat": "stash"}).Error("Can't create ", patFile, " due to ", err)
		return err
	}
	bytesWritten, werr := f.WriteString(pat)
	if werr != nil {
		log.WithFields(log.Fields{"pat": "stash"}).Error("Can't write to ", patFile, " due to ", err)
		return werr
	}
	f.Sync()
	log.WithFields(log.Fields{"pat": "stash"}).Debug("Stashed GitHub Personal Access Token file ", patFile, ", ", bytesWritten, " Bytes written to disk.")
	return nil
}

// Grabs the necessary parameter (GitHub personal access token, owner and repo)
// from environment, if present at all. Note: the CLI arguments will overwrite
// these environment variables
func grabEnv() {
	pubnode = os.Getenv("DPLOY_PUBLIC_NODE")
	pat = os.Getenv("DPLOY_OBSERVER_GITHUB_PAT")
	owner = os.Getenv("DPLOY_OBSERVER_GITHUB_OWNER")
	repo = os.Getenv("DPLOY_OBSERVER_GITHUB_REPO")
	if dr := os.Getenv("DPLOY_OBSERVER_DELAY_REGISTRATION"); dr != "" {
		rd, _ := strconv.ParseUint(dr, 10, 64)
		registerDelay = time.Duration(rd)
	}
	if tb := os.Getenv("DPLOY_OBSERVER_TARGETBRANCH"); tb != "" {
		targetBranch = tb
	}
}

// Authenticates user against repo
// Based on https://godoc.org/github.com/google/go-github/github
func auth() {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: pat})
	log.WithFields(log.Fields{"auth": "step"}).Debug("Token Source ", ts)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	log.WithFields(log.Fields{"auth": "step"}).Debug("Auth client ", tc)
	client = github.NewClient(tc)
	log.WithFields(log.Fields{"auth": "done"}).Debug("GitHub client ", client)
	fmt.Printf("Authentication against GitHub done\n")
}

func marathonURL() string {
	loc := ""
	mesosdns := "http://leader.mesos:8123"
	log.WithFields(log.Fields{"sd": "step"}).Debug("Trying to query HTTP API of ", mesosdns)
	lookup := "marathon.mesos."
	resp, err := http.Get(mesosdns + "/v1/hosts/" + lookup)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Can't look up Marathon address due to ", err)
		return loc
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Error reading response from Mesos-DNS ", err)
		return loc
	}
	var hrecords []HostRecord
	err = json.Unmarshal(body, &hrecords)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Error decoding JSON object ", err)
		return loc
	}
	loc = "http://" + hrecords[0].IP + ":8080"
	log.WithFields(log.Fields{"sd": "done"}).Debug("Found Marathon at ", loc)
	return loc
}

func whereAmI() string {
	loc := pubnode
	mesosdns := "http://leader.mesos:8123"
	log.WithFields(log.Fields{"sd": "step"}).Debug("Trying to query HTTP API of ", mesosdns)
	lookup := "_dploy-observer._tcp.marathon.mesos."
	resp, err := http.Get(mesosdns + "/v1/services/" + lookup)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Can't look up my address due to ", err)
		return loc
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Error reading response from Mesos-DNS ", err)
		return loc
	}
	var srvrecords []SRVRecord
	err = json.Unmarshal(body, &srvrecords)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Error decoding JSON object ", err)
		return loc
	}
	loc = "http://" + pubnode + ":" + srvrecords[0].Port
	log.WithFields(log.Fields{"sd": "done"}).Debug("Found myself at ", loc)
	return loc
}

// Checks if a Webhook already exists
func hookExists() (bool, int) {
	opt := &github.ListOptions{Page: 1}
	hooks, _, err := client.Repositories.ListHooks(owner, repo, opt)
	if err != nil {
		log.WithFields(log.Fields{"hook": "check"}).Error("Can't query hooks due to ", err)
		return false, 0
	}
	for _, hook := range hooks {
		log.WithFields(log.Fields{"hook": "check"}).Debug("Looking at hook ", hook)
		url, _ := hook.Config["url"].(string)
		if strings.HasSuffix(url, "/dploy") {
			return true, *hook.ID
		}
	}
	return false, 0
}

// Registers a Webhook using https://developer.github.com/v3/repos/hooks
func registerHook() string {
	deployURL := pubnode
	if exists, _ := hookExists(); !exists {
		deployURL = whereAmI() + "/dploy"
		deployHook = new(github.Hook)
		hookType := "web"
		deployHook.Name = new(string)
		deployHook.Name = &hookType
		deployHook.Config = make(map[string]interface{})
		deployHook.Config["url"] = deployURL
		enableHook := true
		deployHook.Active = new(bool)
		deployHook.Active = &enableHook

		// see https://github.com/google/go-github/blob/master/github/repos_hooks.go
		// for details on WebHookPayload
		log.WithFields(log.Fields{"observe": "register"}).Debug("Hook: ", deployHook)
		whp, _, err := client.Repositories.CreateHook(owner, repo, deployHook)
		if err != nil {
			log.WithFields(log.Fields{"observe": "register"}).Debug("Can't register due to: ", err)
			return fmt.Sprintf("Can't register hook due to %s", err)
		}
		log.WithFields(log.Fields{"observe": "done"}).Debug("Registered WebHook ", whp)
		return fmt.Sprintf("Registered WebHook with %s ", string(deployURL))
	} else {
		log.WithFields(log.Fields{"observe": "done"}).Debug("WebHook was already registered!")
		return fmt.Sprintf("WebHook was already registered!")
	}
}

func unregisterHook() string {
	if exists, hid := hookExists(); exists {
		log.WithFields(log.Fields{"observe": "unregister"}).Debug("Hook with ID ", hid)
		_, err := client.Repositories.DeleteHook(owner, repo, hid)
		if err != nil {
			log.WithFields(log.Fields{"observe": "unregister"}).Debug("Can't unregister due to: ", err)
			return fmt.Sprintf("Can't unregister hook due to %s", err)
		}
	}
	return fmt.Sprintf("Unregistered hook")
}

// from http://stackoverflow.com/questions/20357223/easy-way-to-unzip-file-with-golang
// patched f.Mode() -> 0755
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()
		fpath := filepath.Join(dest, f.Name)
		log.WithFields(log.Fields{"observe": "unzip"}).Debug("Extracting ", fpath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, 0755)
			// os.MkdirAll(fpath, f.Mode())
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}
			err = os.MkdirAll(fdir, 0755)
			// err = os.MkdirAll(fdir, f.Mode())
			if err != nil {
				log.Fatal(err)
				return err
			}
			f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
			// f, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func pull(owner, repo, workdir string) error {
	if owner == "" || repo == "" {
		return fmt.Errorf("Don't know where to pull from since no owner or repo set")
	}
	repoURL := "https://github.com/" + owner + "/" + repo + "/archive/" + targetBranch + ".zip"
	_, err := dploy.Download(repoURL, workdir)
	if err != nil {
		log.WithFields(log.Fields{"observer": "pull"}).Error("Failed to download repo content due to ", err)
		return fmt.Errorf("Failed to download repo content due to %s", err)
	}
	log.WithFields(log.Fields{"observe": "pull"}).Debug("Downloaded ", repoURL, " into ", workdir)
	td, _ := filepath.Abs(filepath.Join(workdir, ""))
	if _, err := os.Stat(td); os.IsExist(err) {
		os.RemoveAll(td)
	}
	uerr := unzip(targetBranch+".zip", td)
	if uerr != nil {
		return uerr
	}
	log.WithFields(log.Fields{"observe": "pull"}).Debug("Extracted ", targetBranch+".zip", " into ", td)
	return nil
}

func patchMarathon(workdir string) error {
	ad, _ := filepath.Abs(filepath.Join(workdir, dploy.APP_DESCRIPTOR_FILENAME))
	log.WithFields(log.Fields{"observer": "patchmarathon"}).Debug("Trying to read app descriptor ", ad)
	d, err := ioutil.ReadFile(ad)
	if err != nil {
		return fmt.Errorf("Failed to read app descriptor due to", err)
	}
	appDescriptor := dploy.DployApp{}
	uerr := yaml.Unmarshal([]byte(d), &appDescriptor)
	if uerr != nil {
		log.WithFields(log.Fields{"observer": "patchmarathon"}).Error("Failed to de-serialize app descriptor due to ", uerr)
		return fmt.Errorf("Failed to de-serialize app descriptor due to ", uerr)
	}
	log.WithFields(log.Fields{"observer": "patchmarathon"}).Debug("Got valid app descriptor ")
	appDescriptor.MarathonURL = marathonURL()
	ob, merr := yaml.Marshal(&appDescriptor)
	if merr != nil {
		log.WithFields(log.Fields{"observer": "patchmarathon"}).Error("Failed to serialize app descriptor due to ", merr)
		return fmt.Errorf("Failed to serialize app descriptor due to ", merr)
	}
	f, perr := os.Create(ad)
	if perr != nil {
		log.WithFields(log.Fields{"observer": "patchmarathon"}).Error("Can't create ", ad, " due to ", perr)
		return fmt.Errorf("Failed to serialize app descriptor due to ", perr)
	}
	bytesWritten, err := f.Write(ob)
	f.Sync()
	log.WithFields(log.Fields{"observer": "patchmarathon"}).Debug("Patched ", ad, ", ", bytesWritten, " Bytes written to disk.")
	return nil
}

func bootstrap() {
	log.WithFields(log.Fields{"bootstrap": "step"}).Debug("Starting bootstrap process ...")
	time.Sleep(time.Second * DEFAULT_OBSERVER_WAIT_TIME) // wait for Mesos-DNS to kick in
	log.WithFields(log.Fields{"bootstrap": "step"}).Debug("Waited long enough now for Mesos-DNS, registering Webhook")
	result := registerHook()
	log.WithFields(log.Fields{"bootstrap": "step"}).Debug(result)
	fmt.Printf("%s\n", result)
}

func main() {
	log.SetLevel(log.DebugLevel)
	fmt.Printf("This is dploy observer version %s\n", VERSION)
	fmt.Printf("I'm observing branch %s of repo %s/%s trying to serve on node %s\n", targetBranch, owner, repo, pubnode)
	auth()
	go bootstrap()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")

		s := &Status{
			Owner:        owner,
			Repo:         repo,
			TargetBranch: targetBranch,
			Pubnode:      pubnode,
			LastDeploy:   lastDeployment,
		}
		sb, _ := json.Marshal(s)
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, string(sb))
	})
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		result := registerHook()
		fmt.Printf("Webhook registered\n")
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, `{"result":"`+result+`"}`)
	})
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		result := unregisterHook()
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, `{"result":"`+result+`"}`)
	})
	mux.HandleFunc("/dploy", func(w http.ResponseWriter, r *http.Request) {
		dr := &DployResult{}
		cwd, _ := os.Getwd()
		log.WithFields(log.Fields{"handle": "/dploy"}).Info("Noticed change of branch ", targetBranch, " in ", owner, "/", repo)
		err := pull(owner, repo, cwd)
		if err != nil {
			dr.Success = false
			dr.Msg = fmt.Sprintf("Not able to pull new version of %s/%s due to %v", owner, repo, err)
			drb, _ := json.Marshal(dr)
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, string(drb))
			return
		}
		log.WithFields(log.Fields{"handle": "/dploy"}).Info("Pulled new version, ready to patch Marathon")
		perr := patchMarathon(repo + "-" + targetBranch)
		if perr != nil {
			dr.Success = false
			dr.Msg = fmt.Sprintf("Not able to patch Marathon URL due to %v", perr)
			drb, _ := json.Marshal(dr)
			w.Header().Set("Content-Type", "application/javascript")
			fmt.Fprint(w, string(drb))
			return
		}
		log.WithFields(log.Fields{"handle": "/dploy"}).Info("Patched Marathon, ready to update using workspace " + repo + "-" + targetBranch)
		success := dploy.Upgrade(repo + "-" + targetBranch)
		if success {
			log.WithFields(log.Fields{"handle": "/dploy"}).Info("Update successfully carried out")
		} else {
			log.WithFields(log.Fields{"handle": "/dploy"}).Info("Update problems")
		}
		lastDeployment = time.Now()
		dr.Success = success
		dr.Msg = fmt.Sprintf("New version of %s/%s deployed at %s", owner, repo, lastDeployment)
		drb, _ := json.Marshal(dr)
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, string(drb))

	})
	log.Fatal(http.ListenAndServe(":8888", mux))
}
