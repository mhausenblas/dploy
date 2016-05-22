# dploy

[![Join the chat at https://gitter.im/mhausenblas/dploy](https://badges.gitter.im/mhausenblas/dploy.svg)](https://gitter.im/mhausenblas/dploy?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

The [DC/OS](https://dcos.io) deployment tool for appops allows you to create, deploy and manage services and apps composed of microservices (µS):

- It is simple: it has 8 commands and that's that.
- It is stateless: state is exclusively kept in (local) descriptor and spec files (a collection of Marathon app specs).
- It is self-contained: written in Go, `dploy` is a single binary incl. all dependencies.

## Installation

From source (will get you always the latest version):

    $ go get github.com/mhausenblas/dploy

Alternatively, you can [download binaries](https://github.com/mhausenblas/dploy/releases/tag/0.8.2) for Linux, Mac OS X, and Windows or use it via Docker:

    $ docker pull mhausenblas/dploy:0.8.2
    
    # you can use dploy as follows then (note the usage of the volume):
    $ docker run --rm -it -v /Users/mhausenblas/tmp:/tmp mhausenblas/dploy:0.8.2 -workspace=/tmp init

## Dependencies

- [DC/OS 1.7](https://dcos.io/releases/1.7.0/)
- [github.com/gambol99/go-marathon](https://github.com/gambol99/go-marathon), an API library for working with Marathon.
- [github.com/Sirupsen/logrus](https://github.com/Sirupsen/logrus), a logging library.
- [github.com/olekukonko/tablewriter](https://github.com/olekukonko/tablewriter), a ACSII table formatter.
- [github.com/google/go-github/github](https://godoc.org/github.com/google/go-github/github), a GitHub library.

## Features

- [x] `dploy init` … creates a new µS-based app
- [x] `dploy dryrun` … validates deployment of the µS-based app
- [x] `dploy run`… launches the µS-based app using the Marathon API
- [x] `dploy destroy`… tears down µS-based app using the Marathon API
- [x] `dploy ls` … lists the resources of the µS-based app
- [x] `dploy ps` … lists runtime properties of the µS-based app
- [x] `dploy scale`… scales the µS-based app
- [x] Support push-to-deploy, see [observer](observer/)
- [ ] Add examples (blog2go, rolling upgrades, etc.)
- [ ] Expose metrics via `dploy -all ps`
- [ ] Transparent handling of secrets with [Vault](https://github.com/brndnmtthws/vault-dcos)
- [ ] Advanced µS examples using linkerd, VAMP
- [ ] `dploy publish` … publishes the the µS-based app into the DC/OS Universe
- [ ] Enable tracing using Zipkin: [Mesos scheduler](http://elodina.github.io/zipkin-mesos-framework/) and [Go lib](http://elodina.github.io/go-zipkin/)

## Docs

To view the `dploy` package docs locally, do the following in your Go workspace:

    $ mkdir -p $GOPATH/github.com/mhausenblas/ && cd $GOPATH/github.com/mhausenblas/
    $ git clone https://github.com/mhausenblas/dploy.git && cd dploy
    $ godoc -http=":6060"

If you now visit [http://localhost:6060/pkg/github.com/mhausenblas/dploy/lib/](http://localhost:6060/pkg/github.com/mhausenblas/dploy/lib/) in your favorite Web browser you should be able to see the `dploy` package docs:

![Docs for dploy](img/dploy_godocs.png)