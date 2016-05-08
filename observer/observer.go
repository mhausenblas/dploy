package main

import (
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	github "github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"strings"
)

const (
	VERSION        string = "0.2.0"
	OBSERVE_BRANCH string = "dcos"
)

var (
	// the HTTP request multiplexer
	mux *http.ServeMux

	// the GitHub client
	client *github.Client

	// personal access token (pat), owner and repo to observe
	pat, owner, repo string

	// Web hook used to trigger deployment
	deployHook *github.Hook
)

type DployResult struct {
	success bool
	msg     string
}

func init() {
	mux = http.NewServeMux()
	grabEnv() // try via env variables first
	flag.StringVar(&pat, "pat", "", "the personal access token, via https://github.com/settings/tokens")
	flag.StringVar(&owner, "owner", "", "the GitHub owner, for example 'mhausenblas' or 'mesosphere'.")
	flag.StringVar(&repo, "repo", "", "the GitHub repo, for example 'dploy' or 'marathon'.")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
}

// Grabs the necessary parameter (GitHub personal access token, owner and repo)
// from environment, if present at all. Note: the CLI arguments will overwrite
// these environment variables
func grabEnv() {
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
}

// Checks if a Webhook already exists
func hookExists() bool {
	opt := &github.ListOptions{Page: 1}
	hooks, _, err := client.Repositories.ListHooks(owner, repo, opt)
	if err != nil {
		log.Fatal("Can't query hooks due to %v", err)
	}
	for _, hook := range hooks {
		log.WithFields(log.Fields{"hook": "check"}).Debug("Looking at hook:\n", hook)
		url, _ := hook.Config["url"].(string)
		if strings.HasSuffix(url, "/dploy") {
			return true
		}
	}
	return false
}

// Registers a Webhook using https://developer.github.com/v3/repos/hooks
func registerHook() {

	//TODO: service discovery (via Mesos-DNS): on which node/port am I serving and put that into Config["url"]
	if !hookExists() {
		deployHook = new(github.Hook)
		hookType := "web"
		deployHook.Name = new(string)
		deployHook.Name = &hookType
		deployHook.Config = make(map[string]interface{})
		deployHook.Config["url"] = "http://localhost:8888/dploy"
		enableHook := true
		deployHook.Active = new(bool)
		deployHook.Active = &enableHook

		// see https://github.com/google/go-github/blob/master/github/repos_hooks.go
		// for details on WebHookPayload
		log.WithFields(log.Fields{"observe": "register"}).Debug("Hook: ", deployHook)
		whp, _, err := client.Repositories.CreateHook(owner, repo, deployHook)
		if err != nil {
			log.Fatal("Can't register hook due to %v", err)
		}
		log.WithFields(log.Fields{"observe": "done"}).Debug("Registered WebHook ", whp)
	} else {
		log.WithFields(log.Fields{"observe": "done"}).Debug("WebHook was already registered!")
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	fmt.Printf("Observing the %s branch of %s/%s\n", OBSERVE_BRANCH, owner, repo)
	auth()
	registerHook()
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"status":"ok"}`)
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
