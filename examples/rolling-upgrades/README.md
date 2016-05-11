# Rolling upgrades examples

To do rolling upgrades with dploy, you use a declarative approach. Based on Marathon's [upgrade strategies](https://mesosphere.github.io/marathon/docs/) you define what upgrade strategy to apply for a µS.
In the following I'll walk you through a couple of zero-downtime scenarios, from the simple case of a rolling upgrade to blue-green deployments to canary releases.

Requirements (in addition to a running DC/OS 1.7 cluster):

- The [DC/OS CLI](https://dcos.io/docs/1.7/...) is installed
- The [jq](http://xxx) tool is installed (for querying JSON data)

## A simple rolling upgrade

In the [simplest case](simple-0downtime.json) of a rolling upgrade, use the following upgrade strategy:

```javascript
...
"upgradeStrategy": {
	"minimumHealthCapacity": 0.85,
	"maximumOverCapacity": 0.15
},
...
```

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