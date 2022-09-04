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
