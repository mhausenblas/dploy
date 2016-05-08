package main

import (
	"flag"
	"fmt"
	log "github.com/Sirupsen/logrus"
	github "github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

const (
	OBSERVE_BRANCH     string        = "dcos"
	DEFAULT_POLL_DELAY time.Duration = 1
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

func init() {
	mux = http.NewServeMux()

	flag.StringVar(&pat, "pat", "", "the personal access token, via https://github.com/settings/tokens")
	flag.StringVar(&owner, "owner", "", "the GitHub owner, for example 'mhausenblas' or 'mesosphere'.")
	flag.StringVar(&repo, "repo", "", "the GitHub repo, for example 'dploy' or 'marathon'.")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
}

// Authenticates user against repo
// Ripped off of https://github.com/google/go-github/blob/master/examples/basicauth/main.go
func auth() {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: pat})
	log.WithFields(log.Fields{"auth": "step"}).Debug("Token Source ", ts)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	log.WithFields(log.Fields{"auth": "step"}).Debug("Auth client ", tc)
	client = github.NewClient(tc)
	log.WithFields(log.Fields{"auth": "done"}).Debug("GitHub client ", client)
}

// Registers a WebHook using https://developer.github.com/v3/repos/hooks
func registerHook() {
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
		fmt.Fprint(w, `{"status":"dploy run"}`)
	})
	log.Fatal(http.ListenAndServe("localhost:8888", mux))
}