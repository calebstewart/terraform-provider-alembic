---
page_title: "Provider: Alembic"
subcategory: ""
description: |-
  Terraform provider for automatic database schema upgrades with [Alembic](https://alembic.sqlalchemy.org/en/latest/).
---

# Alembic Provider

This provider allows you to easily embed your Alembic database migration execution
into your terraform deployment scripts.

**Note**: The extra arguments specified at the provider level are always passed in
addition to any extra arguments specified at the resource level. 

## Example Usage

```terraform
provider "alembic" {
  project_root = "${path.module}/../"
  alembic = ["poetry", "run", "alembic"]
}
```

## Schema

### Required

- **project_root** (String) Path to the root of your project where `alembic.ini` can be found.

### Optional

- **alembic** (List[String], Optional) The command used to invoke Alembic (default: `["alembic"]`)
- **config** (String, Optional) Name of the alembic configuration file (default: `alembic.ini`)
- **section** (String, Optional) Section within the configuration to use for Alembic config (default: `alembic`)
- **extra** (Map[String]String, Optional) Additional arguments consumed by custom `env.py` scripts 

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
