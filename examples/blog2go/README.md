# Blog2Go example

In this example, we will walk through setting up and using Jekyll, a popular static blogging platform, via dploy.

## Preparation

You'll need a `dploy.app` app descriptor with [a content](https://github.com/mhausenblas/s4d/tree/blog2go) looking like the following (replace with your own values):

```
marathon_url: http://localhost:8080
app_name: myblog
repo_url: https://github.com/mhausenblas/s4d
public_node: 52.24.105.248
trigger_branch: blog2go
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
$ BLOG2GO_IP=$(echo "curl -s ifconfig.ca" | dcos node ssh --master-proxy --mesos-id=$(dcos task --json | jq --raw-output '.[] | select(.name == "blog2go.dployex") | .slave_id') 2>/dev/null) 
$ BLOG2GO_PORT=$(dcos marathon task list --json | jq "map(select(.appId==\"/dployex/blog2go\").ports[0])" | tail -2 | head -n 1 | cut -c 3-)
$ BLOG2GO_URL=http://$BLOG2GO_IP:$BLOG2GO_PORT/
$ echo $BLOG2GO_URL
```
Now, if you visit the value of `$BLOG2GO_URL` in your browser (output of last line of above) you should see the following:

TODO: insert screen shot here

## Publishing posts

Now edit XXX to create a new post and then do:

```bash
$ git add site/*
$ git commit -m "Publishing my first blog post"
$ git push
```

TODO: insert screen shot here