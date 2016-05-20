# Examples of stateful apps

In the following we will walk through a number of stateful apps, starting with Wordpress (with a MySQL backend).

## Wordpress

Make sure you've got this example installed, either by using `DPLOY_EXAMPLES=all dploy` or by manually downloading the [Marathon app spec](wordpress.json) into your `specs/` directory. This example uses [Marathon-lb](https://dcos.io/docs/1.7/usage/) as an edge router so you'll have to adapt the following line in `specs/wordpress.json`:

    "HAPROXY_0_VHOST": "$MASTERFQDN"

Replace `$MASTERFQDN` with the DNS name of your DC/OS Dashboard, something like `test-elasticlo-15yui6ie73cqf-105895761.us-west-2.elb.amazonaws.com` (note: without `https://`) and then you can do the following:

    $ dploy dryrun
    $ dploy run

Now, go to your public node and use port `10000` to access the app:

TODO: insert screen shot here

Note that this is a test/dev setup since it doesn't use persistent or external volumes, which means that if the container dies all the data is lost.