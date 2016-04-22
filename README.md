# dploy, the DC/OS deployment tool

dploy the DC/OS deployment tool for appops.

## Commands

- `dploy init` … creates a new app for you, that is a `dploy.app` file is created in current dir with default values
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