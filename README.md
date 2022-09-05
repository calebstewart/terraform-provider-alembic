# Terraform Provider Alembic

This provider makes it possible to integrate Alembic database migrations into
your Terraform deployment flow natively. In the past, I've used external Python
scripts to do this, but that has the downside of invoking the script on state
updates, which is not ideal for things like Alembic which may modify the
databae.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.18

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

See the generated documentation for usage examples. You will need a project
using SQLAlchemy and Alembic as well as a database instance which is accessible
from the host running terraform (optionally through a proxy of some sort).

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
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
