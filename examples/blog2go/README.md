# Blog2Go example

In this example, we will walk through setting up and using Jekyll, a popular static blogging platform, via dploy.


Via volume mount + uris:

	"volumes": [
		{
			"containerPath": "/srv/jekyll",
			"hostPath": "$(pwd)",
			"mode": "RW"
		}
	],
    "uris": ...

## Preparation

You'll need a `dploy.app` app descriptor with a content looking like the following (replace with your own values):

```
marathon_url: http://localhost:8080
app_name: blog2go
repo_url: https://github.com/mhausenblas/s4d
public_node: 52.24.105.248
trigger_branch: master
```

In addition you'll need a GitHub Personal Access Token and make it available via a `.pat` file, see the [observer docs](../../observer/) for details.

## Launch

To launch the blog, do:

```bash
$ dploy dryrun
$ dploy run
$ dploy ps
```

To find out where your blog is serving and available on the public Internet, do the following (for AWS/CoreOS):

```bash
$ echo "curl -s ifconfig.co" | dcos node ssh --master-proxy --mesos-id=$(dcos task --json | jq --raw-output '.[] | select(.name == "/dployex/blog2go") | .slave_id') 2>/dev/null
```

TODO: insert screen shot here

## Publishing posts

Now edit XXX to create a new post and then do:

```bash
$ git add site/*
$ git commit -m "Publishing my first blog post"
$ git push
```

TODO: insert screen shot here