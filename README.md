# dploy, the DC/OS deployment tool

dploy the DC/OS deployment tool for appops.

## Dependencies

- [DC/OS 1.7](https://dcos.io/releases/1.7.0/)
- [github.com/gambol99/go-marathon](https://github.com/gambol99/go-marathon), an API library for working with Marathon.
- [github.com/Sirupsen/logrus](https://github.com/Sirupsen/logrus), a logging library.

## Usage

### Installation

From source:

    $ go get github.com/mhausenblas/dploy
    $ go build

Download binaries:

    TBD

As a preparation to deal with the JWT Auth, you can simply create an SSH tunnel to the DC/OS Master:

    $ ssh -i ~/.ssh/MYKEY core@MYMASTER -f -L 8080:localhost:8080 -N

From here on, use `http://localhost:8080` for the Marathon base URL.

### Workflow

- `dploy init` … creates a new app for you, that is, a `dploy.app` file with default values is created in `-location`
- `dploy dryrun` … validates deployment of the app by checking if DC/OS cluster is valid, validates app specs, etc.
- `kploy run`… launches the app using the Marathon API
- `dploy ls` … lists the content of the app, all its resources such as Marathon app specs, etc.
- `dploy ps` … lists runtime properties of the app, such as on which node/port its running, etc.
- `dploy update`… lets you update properties of the app, such as scaling or environment variables
- `dploy destroy`… tears down your app
- `dploy publish` … publishes the app in the DC/OS Universe

### dploy init
### dploy dryrun
### dploy run
### dploy ls
### dploy ps
### dploy update
### dploy destroy
### dploy publish