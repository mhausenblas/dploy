package main

import (
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	github "github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	VERSION        string = "0.6.0"
	OBSERVE_BRANCH string = "dcos"
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
)

type DployResult struct {
	success bool
	msg     string
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

func init() {
	mux = http.NewServeMux()
	grabEnv() // try via env variables first
	flag.StringVar(&pat, "pat", pat, "the personal access token, via https://github.com/settings/tokens")
	flag.StringVar(&owner, "owner", owner, "the GitHub owner, for example 'mhausenblas' or 'mesosphere'.")
	flag.StringVar(&repo, "repo", repo, "the GitHub repo, for example 'dploy' or 'marathon'.")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
}

// Grabs the necessary parameter (GitHub personal access token, owner and repo)
// from environment, if present at all. Note: the CLI arguments will overwrite
// these environment variables
func grabEnv() {
	pubnode = os.Getenv("DPLOY_PUBLIC_NODE")
	pat = os.Getenv("DPLOY_OBSERVER_GITHUB_PAT")
	owner = os.Getenv("DPLOY_OBSERVER_GITHUB_OWNER")
	repo = os.Getenv("DPLOY_OBSERVER_GITHUB_REPO")
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

func whereAmI() string {
	loc := pubnode
	mesosdns := "http://leader.mesos:8123"
	log.WithFields(log.Fields{"sd": "step"}).Debug("Trying to query HTTP API of ", mesosdns)
	lookup := "_dploy-observer._tcp.marathon.mesos."
	resp, err := http.Get(mesosdns + "/v1/services/" + lookup)
	if err != nil {
		log.WithFields(log.Fields{"sd": "step"}).Error("Can't look up my address due to error ", err)
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

func main() {
	log.SetLevel(log.DebugLevel)
	fmt.Printf("This is dploy observer version %s\n", VERSION)
	fmt.Printf("I'm observing branch %s of repo %s/%s trying to serve on node %s\n", OBSERVE_BRANCH, owner, repo, pubnode)
	auth()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, `{"status":"ok"}`)
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
		//pull(owner, repo)
		//success := dploy.Run(os.Getwd(), false)
		dr := DployResult{}
		dr.success = true
		dr.msg = "ok"
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, json.NewEncoder(w).Encode(dr))
	})
	log.Fatal(http.ListenAndServe(":8888", mux))
}
