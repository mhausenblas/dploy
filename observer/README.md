# The push-to-deploy service observer

In v0.8 a new, exciting feature has been introduced: *push-to-deploy*. Via the `observer` sub-component `dploy` is now able to upgrade a running version of your app triggered by a `git push`.

To enable the *push-to-deploy* feature, simply add two more attributes to the descriptor file `dploy.app`:

- `repo_url` … defining the repo you want to push to, for example `https://github.com/mhausenblas/s4d`
- `public_node` … the IP address or FQDN of the public DC/OS node

So a complete `dploy.app` example content might look as follows:

    marathon_url: http://localhost:8080
    app_name: mh9test
    repo_url: https://github.com/mhausenblas/s4d
    public_node: 52.37.239.156
    trigger_branch: master

What happens is that with these two additional attributes, `dploy` registers a GitHub [Webhook](https://developer.github.com/webhooks/) the first time you run `dploy run`. From then on you can upgrade your app using  `git push`. Note that the `observer` is by default looking at the `dcos` branch but you can overwrite this using `trigger_branch` as an additional (optional) attribute in the descriptor file (last line of above YAML file).

However, in order to make this work, an additional piece of data (a secret token) is necessary: a GitHub Personal Access Token (PAT). So, go to [github.com/settings/tokens](https://github.com/settings/tokens) and create a token. Let's say the token's value is `123abc*&%xzy`. Copy this token and paste it into a file called `.pat` in the home directory of the Git repo; for example if the GitHub repo is [mhausenblas/s4d](https://github.com/mhausenblas/s4d) then this is what I'd expect to see on my local machine after cloning it:

```bash
~/Documents/repos/mhausenblas/s4d (master)$ ls -al
total 64
drwxr-xr-x   4 mhausenblas  staff    306 10 May 17:06 .
drwxr-xr-x  26 mhausenblas  staff    918  8 May 10:59 ..
-rw-r--r--@  1 mhausenblas  staff   6148  8 May 11:01 .DS_Store
drwxr-xr-x   8 mhausenblas  staff    510 10 May 17:13 .git
-rw-r--r--@  1 mhausenblas  staff      4 10 May 11:50 .gitignore
-rw-r--r--   1 mhausenblas  staff     41 10 May 11:49 .pat
-rw-r--r--   1 mhausenblas  staff  11357  8 May 10:59 LICENSE
-rw-r--r--@  1 mhausenblas  staff    149 10 May 15:58 dploy.app
drwxr-xr-x   2 mhausenblas  staff    136  9 May 19:39 specs

~/Documents/repos/mhausenblas/s4d (master)$ cat .pat
123abc*&%xzy
```

Also, note since the GitHub Personal Access Token is a powerful, security-critical piece of data, you don't want to check it into the repo itself: add `.pat` to the `.git-ignore` file!


## Development

NOTE: the following is only interesting and necessary for dploy developers, not users.

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
$ DPLOY_PUBLIC_NODE=1.2.3.4 observer -pat=4**************************************c -owner=mhausenblas -repo=s4d
Observing the dcos branch of mhausenblas/s4d
DEBU[0000] Token Source {0xc8200621e0}                   auth=step
DEBU[0000] Auth client &{0xc820015080 <nil> <nil> 0}     auth=step
DEBU[0000] GitHub client &{0xc8200150b0 {0 0} https://api.github.com/ https://uploads.github.com/ go-github/0.1 {0 0} [{0 0 {{0 0 <nil>}}} {0 0 {{0 0 <nil>}}}] 0 0xc82002c028 0xc82002c030 0xc82002c038 0xc82002c040 0xc82002c048 0xc82002c050 0xc82002c058 0xc82002c060 0xc82002c068 0xc82002c070 0xc82002c078 0xc82002c080 0xc82002c088}  auth=done
DEBU[0000] Hook: github.Hook{Name:"web", Active:true, Config:map[url:http://localhost:8888/dploy]}  observe=register
DEBU[0000] Payload github.Hook{CreatedAt:time.Time{sec:, nsec:, loc:time.Location{name:"UTC", cacheStart:, cacheEnd:}}, UpdatedAt:time.Time{sec:, nsec:, loc:time.Location{name:"UTC", cacheStart:, cacheEnd:}}, Name:"web", URL:"https://api.github.com/repos/mhausenblas/s4d/hooks/8319869", Events:["push"], Active:true, Config:map[url:http://localhost:8888/dploy], ID:8319869}
```

The `observer` service is meant to be used as a containerized service via Marathon, configured and launched by the main `dploy` command and not separately and/or manually.

For `dploy` being able to launch the configured `observer` instance, two things are necessary: on the one hand a dedicated `observer` [Dockerfile](Dockerfile) (see also the corresponding Docker [image](https://hub.docker.com/r/mhausenblas/dploy-observer/)), and on the other hand a [Marathon app spec template](observer.json) that requires `dploy` to set the following run-time parameters:

- `DPLOY_PUBLIC_NODE` ... the IP address or FQDN of the public node
- `DPLOY_OBSERVER_GITHUB_PAT` ... the GitHub personal access token, needs to be manually created beforehand via https://github.com/settings/tokens
- `DPLOY_OBSERVER_GITHUB_OWNER` ... the GitHub owner (handle or profile) to observe
- `DPLOY_OBSERVER_GITHUB_REPO` ... the GitHub repo to observe

Note that the last three parameters are exposed as environment variables in the Marathon app spec template (could also be provided via arguments).

Once launched, the output of the `observer` service in DC/OS (Mesos view, drilling down to the task sandbox) should be something like the following.

On`stdout`:

```bash
--container="mesos-d54d3818-dc7e-423f-ba92-1238ed35eecf-S0.cf1d9ff0-b644-477b-b2c3-26f0925c63d2" --docker="docker" --docker_socket="/var/run/docker.sock" --help="false" --initialize_driver_logging="true" --launcher_dir="/opt/mesosphere/packages/mesos--cdba65a401eec9e5583daaa84fb10c91d2373d51/libexec/mesos" --logbufsecs="0" --logging_level="INFO" --mapped_directory="/mnt/mesos/sandbox" --quiet="false" --sandbox_directory="/var/lib/mesos/slave/slaves/d54d3818-dc7e-423f-ba92-1238ed35eecf-S0/frameworks/d54d3818-dc7e-423f-ba92-1238ed35eecf-0000/executors/dploy-observer.65e5738f-154d-11e6-864f-1ad90584ec17/runs/cf1d9ff0-b644-477b-b2c3-26f0925c63d2" --stop_timeout="0ns"
--container="mesos-d54d3818-dc7e-423f-ba92-1238ed35eecf-S0.cf1d9ff0-b644-477b-b2c3-26f0925c63d2" --docker="docker" --docker_socket="/var/run/docker.sock" --help="false" --initialize_driver_logging="true" --launcher_dir="/opt/mesosphere/packages/mesos--cdba65a401eec9e5583daaa84fb10c91d2373d51/libexec/mesos" --logbufsecs="0" --logging_level="INFO" --mapped_directory="/mnt/mesos/sandbox" --quiet="false" --sandbox_directory="/var/lib/mesos/slave/slaves/d54d3818-dc7e-423f-ba92-1238ed35eecf-S0/frameworks/d54d3818-dc7e-423f-ba92-1238ed35eecf-0000/executors/dploy-observer.65e5738f-154d-11e6-864f-1ad90584ec17/runs/cf1d9ff0-b644-477b-b2c3-26f0925c63d2" --stop_timeout="0ns"
Registered docker executor on 10.0.7.91
Starting task dploy-observer.65e5738f-154d-11e6-864f-1ad90584ec17
This is dploy observer version 0.5.0
I'm observing branch dcos of repo mhausenblas/s4d
Authentication against GitHub done
Webhook registered
Received killTask
Shutting down
```

On `stderr`:

```bash
I0508 18:48:06.216948  7130 exec.cpp:143] Version: 0.28.1
I0508 18:48:06.221783  7137 exec.cpp:217] Executor registered on slave d54d3818-dc7e-423f-ba92-1238ed35eecf-S0
time="2016-05-08T18:48:06Z" level=debug msg="Token Source {0xc82000e300}" auth=step 
time="2016-05-08T18:48:06Z" level=debug msg="Auth client &{0xc820013080 <nil> <nil> 0}" auth=step 
time="2016-05-08T18:48:06Z" level=debug msg="GitHub client &{0xc8200130b0 {0 0} https://api.github.com/ https://uploads.github.com/ go-github/0.1 {0 0} [{0 0 {{0 0 <nil>}}} {0 0 {{0 0 <nil>}}}] 0 0xc82002a030 0xc82002a038 0xc82002a040 0xc82002a048 0xc82002a050 0xc82002a058 0xc82002a060 0xc82002a068 0xc82002a070 0xc82002a078 0xc82002a080 0xc82002a088 0xc82002a090}" auth=done 
time="2016-05-08T18:48:06Z" level=debug msg="Trying to query HTTP API of http://leader.mesos:8123" sd=step 
time="2016-05-08T18:48:06Z" level=debug msg="Found myself at http://52.37.239.156:8849" sd=done 
time="2016-05-08T18:48:06Z" level=debug msg="Hook: github.Hook{Name:\"web\", Active:true, Config:map[url:http://52.37.239.156:8849/dploy]}" observe=register 
time="2016-05-08T18:48:07Z" level=debug msg="Registered WebHook github.Hook{CreatedAt:time.Time{sec:, nsec:, loc:time.Location{name:\"UTC\", cacheStart:, cacheEnd:}}, UpdatedAt:time.Time{sec:, nsec:, loc:time.Location{name:\"UTC\", cacheStart:, cacheEnd:}}, Name:\"web\", URL:\"https://api.github.com/repos/mhausenblas/s4d/hooks/8321735\", Events:[\"push\"], Active:true, Config:map[url:http://52.37.239.156:8849/dploy], ID:8321735}" observe=done 
```

Currently, while the `observer` Marathon app will be removed when user issue the `dploy destroy` command, the Webhooks still requires manual removal.
