# Rolling upgrades examples

To do rolling upgrades with dploy, you use a declarative approach. Based on Marathon's [upgrade strategies](https://mesosphere.github.io/marathon/docs/rest-api.html#upgrade-strategy) you define what upgrade strategy to apply for a µS.
In the following I'll walk you through a couple of zero-downtime scenarios, from the simple case of a rolling upgrade to blue-green deployments to canary releases.

Requirements (in addition to a running DC/OS 1.7 cluster):

- The [DC/OS CLI](https://dcos.io/docs/1.7/...) is installed
- The [jq](http://xxx) tool is installed (for querying JSON data)

## A simple rolling upgrade

In the [simplest case](simple-0downtime.json) of a rolling upgrade, use the following upgrade strategy:

```javascript
...
"upgradeStrategy": {
	"minimumHealthCapacity": 0.25,
	"maximumOverCapacity": 0.25
},
...
```

The meaning of `minimumHealthCapacity`  and `maximumOverCapacity` is as follows:

- `minimumHealthCapacity` …  a floating point value between 0 and 1 (which defaults to `1`), specifying the % of instances to maintain healthy during deployment; with `0` meaning all old instances are stopped before the new version is deployed and `1` meaning all instances of the new version are deployed side by side with the old one before it is stopped.
- `maximumOverCapacity` …  a floating point value between 0 and 1 (which defaults to `1`), specifying the max. % of instances over capacity during deployment; with `0` meaning that during the upgrade process no additional capacity than may be used for old and new instances ( only when an old version is stopped, a new instance can be deployed) and  `1` meaning that all old and new instances can co-exist during the upgrade process.

Note that the default values (both are `1`) mean a safe but somewhat resource-intensive upgrade.

The following example, with `[vN]` being an app instance on version `N`, having a scale factor of `"instances": 4` and assuming a `"minimumHealthCapacity": 0.25` and `"maximumOverCapacity": 0.25` shows what this means in practice:

```
T0:    [v1] [v1] [v1] [v1]     
                               
T1:    deployment kicks off    
                               
T2:    [v1] [v1] [v1] [v1] [v2]
                       |       
T3:    [v1] [v1] [v1] [v2] [v2]
                  |            
T4:    [v1] [v1] [v2] [v2] [v2]
             |                
T5:    [v1] [v2] [v2] [v2] [v2]
        |                      
T6:    [v2] [v2] [v2] [v2]     
                               
T7:    deployment done         
```

A `minimumHealthCapacity` of `0.25` means that 25% (or: exactly one instance in our case) always needs to run on a certain version. I other words, at no time in the deployment can the app have less than one instance running with any given version, say, `v1`. When the deployment kicks off at timepoint `T1` (before that and up to incl. `T0` the current version of the app was `v1`) the `maximumOverCapacity` attribute becomes important: since we've set it to `0.25`  it means no more than 25% (or: exactly one instance in our case) can be run in addition to the already running instances. In other words: with this setting, no more than 5 instances of the app (in whatever version they might be in) can ever run at the same time. At `T2` one instance at version `v2` comes up, satisfying both capacity requirements; at `T3`, one `v1` instance is stopped and replaced by a `v2` instance; at `T4` the same happens again and with the `T5-T6` transition the last remaining `v1` instance is stopped and since we now have 4 instances of `v2` running all is good and as expected at `T7`.

Lesson learned: certain combinations of `minimumHealthCapacity` and `maximumOverCapacity` make sense while others are not satisfiable, meaning that you can specify them, just the deployment will never be carried out. For example, a `"minimumHealthCapacity": 0.5` and `"maximumOverCapacity": 0.1` would be unsatisfiable (since you want to keep at least half of your instances around but only allow 10% overcapacity); to make this deployment satisfiable you'd need to change it to `"maximumOverCapacity": 0.5`.

Initially, we deploy the app using the standard run command (make sure you have enabled [push-to-deploy](../../observer/) for the following to work):

```bash
$ dploy run
```

Above command rolls out the initial version (`v1`) and also registers a Webhook with GitHub that watches future `git push` events.

Now it's time to change [simple-0downtime.json](simple-0downtime.json) to reflect a new version (`v2`) being available for deployment:

```javascript
...
"cmd": "echo <img src=\"https://raw.githubusercontent.com/mhausenblas/dploy/master/examples/rolling-upgrades/res/v2.jpg\"/ alt=\"v2\"> >index.html && python3 -m http.server 8080",
...
```

Once the edits are done, you can kick off the rolling upgrade using `git` like so:

```bash
$ git add simple-0downtime.json
$ git commit -m"Version v2"
$ git push origin master
```

Now we need to figure out where the new instances are serving, so we do some manual service discovery using the DC/OS CLI (requires the `jq` tool to be installed):

```bash
$ dcos marathon task list --json | jq "map(select(.appId==\"/dployex/appserver\").host, select(.appId==\"/dployex/appserver\").ports)"
```

Note that in a real-world setup you'd want to use a load-balancer in front of your app server instances. The load-balancer would then take care of the service discovery for you, in addition to spreading the load across the instances.

## Blue-green deployment

TBD

## Canary releases

TBD