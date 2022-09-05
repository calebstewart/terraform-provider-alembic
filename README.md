# Terraform Provider Alembic

This provider makes it possible to integrate Alembic database migrations into
your Terraform deployment flow natively. In the past, I've used external Python
scripts to do this, but that has the downside of invoking the script on state
updates, which is not ideal for things like Alembic which may modify the
databae.

I hope to support downgrade and stamp resources eventually, but for now I am
focusing on getting the `alembic_upgrade` resource right.

## Build provider

Run the following command to build the provider

```shell
$ go build -o terraform-provider-alembic
```

## Test sample configuration

First, build and install the provider.

```shell
$ make install
```

Then, navigate to the `examples` directory. 

```shell
$ cd examples
```

You will need to add relevant provider configurations to `main.tf` to tell
the provider where your Alembic configuration is located and how to execute
Alembic. Then, run the following command to initialize the workspace and
apply the sample configuration.

```shell
$ terraform init && terraform apply
```

## Note on Proxy Commands

Both the `alembic_upgrade` and `alembic_stamp` resources provide an optional
attribute named `proxy_command` which is an argument list to be executed
for the duration of your Alembic execution. The intent is to start some
form of proxy needed to access your database server directly. In the case
of Google Cloud Platform deployments, this is commonly `cloud_sql_proxy`,
but could be anything that proxies traffic to your database instance. For
example, it could be an SSH command to tunnel traffic to an internal instance.

Because this is a free-form argument list for a process, there are a few
things worth noting:

1. Obviously, this allows execution of arbitrary other binaries through
   your terraform configuration. Care should be taken with what you
   execute through this configuration.
2. There is another optional configuration named `proxy_sleep` which
   controls how long the provider should sleep after invoking the proxy
   command before attempting to execute Alembic commands. This is because
   the proxy may not be up immediately, and attempting to start Alembic
   quickly normally results in connection timeouts. The default value
   of this configuration is `5s` which will cause each operation
   (Read, Update, Create, etc) to sleep for 5 seconds before starting
   execution.
3. The third-party command must obvioulsy be installed on the system
   running terraform. In the case of `cloud_sql_proxy`, that requires
   you to go and download the static binary provided by Google, and
   placing it either in your `PATH` or providing the fully qualified
   path as the first element in your `proxy_command` array.
