# Build pipeline

Currently, there is no automated build pipeline but when I get around I plan to set up a [Jenkins on DC/OS](https://dcos.io/docs/1.7/usage/tutorials/jenkins/) based one. For cutting a release, I currently manually do:

    # to generate the Linux binary:
    $ GOOS=linux go build 
    # to generate the Windows binary:
    $ GOOS=windows go build
    # to generate the OSX binary:
    $ go build

After this, I manually create a new [release](https://github.com/mhausenblas/dploy/releases) and upload those binaries, renaming them accordingly, where necessary.