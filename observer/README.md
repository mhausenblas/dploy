# The push-to-deploy service observer

If a `repo_url` is specified in the app descriptor `dploy.app`, then you can use the `observer` service to: i) automatically register a GitHub [Webhook](https://developer.github.com/webhooks/) for the repo, and ii) trigger a deployment (`dploy run`) every time a `git push` to the `dcos` branch occurs.

Usage (test/development):

```bash
$ observer -h
  -owner string
    	the GitHub owner, for example 'mhausenblas' or 'mesosphere'.
  -pat string
    	the personal access token, via https://github.com/settings/tokens
  -repo string
    	the GitHub repo, for example 'dploy' or 'marathon'.
```

Example:

```bash
$ observer -pat=4**************************************c -owner=mhausenblas -repo=s4d
Observing the dcos branch of mhausenblas/s4d
DEBU[0000] Token Source {0xc8200621e0}                   auth=step
DEBU[0000] Auth client &{0xc820015080 <nil> <nil> 0}     auth=step
DEBU[0000] GitHub client &{0xc8200150b0 {0 0} https://api.github.com/ https://uploads.github.com/ go-github/0.1 {0 0} [{0 0 {{0 0 <nil>}}} {0 0 {{0 0 <nil>}}}] 0 0xc82002c028 0xc82002c030 0xc82002c038 0xc82002c040 0xc82002c048 0xc82002c050 0xc82002c058 0xc82002c060 0xc82002c068 0xc82002c070 0xc82002c078 0xc82002c080 0xc82002c088}  auth=done
DEBU[0000] Hook: github.Hook{Name:"web", Active:true, Config:map[url:http://localhost:8888/dploy]}  observe=register
DEBU[0000] Payload github.Hook{CreatedAt:time.Time{sec:, nsec:, loc:time.Location{name:"UTC", cacheStart:, cacheEnd:}}, UpdatedAt:time.Time{sec:, nsec:, loc:time.Location{name:"UTC", cacheStart:, cacheEnd:}}, Name:"web", URL:"https://api.github.com/repos/mhausenblas/s4d/hooks/8319869", Events:["push"], Active:true, Config:map[url:http://localhost:8888/dploy], ID:8319869}
```

The `observer` service is meant to be used as a containerized service via Marathon:

- the observer Docker image

- the observer.json app spec


`DPLOY_OBSERVER_GITHUB_PAT`, `DPLOY_OBSERVER_GITHUB_OWNER`, and `DPLOY_OBSERVER_GITHUB_REPO`