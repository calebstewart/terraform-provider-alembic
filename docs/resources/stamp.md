---
page_title: "stamp Resource - terraform-provider-alembic"
subcategory: ""
description: |-
  The stamp resource allows you to stamp a database with a specified revision ID without performing
  any direct upgrades.
---

# Resource `alembic_upgrade`

The stamp resource allows you to stamp a database with a specified revision ID
during infrastructure deployment. If environment variables are passed, they are
marked as sensitive, since they likely contain database credentials or
configuration collected from infrastructure deployment.

**Note**: Extra arguments specified at the provider level are always passed in
addition to any extra arguments specified here at the resource level.

## Example Usage

```terraform
resource "alembic_upgrade" "db-upgrade" {
  revision = "head"
  tag      = "my-custom-tag"

  environment = {
    DATABASE_URL = locals.database_connection_string
  }
}
```

## Argument Reference

- `revision` - (String, Required) The name of the target revision for the upgrade.
- `tag` - (String, Optional) An arbitrary tag which can be used by custom `env.py` scripts.
- `environment` - (Map[String, String], Optional, *Sensitive*) Environment variables to set when invoking Alembic.
- `alembic` - (List[String], Optional) The command used to invoke Alembic (default: provider configuration).
- `proxy_command` - (List[String], Optional) A command which is executed for the lifetime of database interactions used to proxy connections for protected instances (e.g. [Cloud SQL Proxy](https://cloud.google.com/sql/docs/mysql/sql-proxy)).
- `extra` - (Map[String]String, Optional) Additional arguments consuemd by custom `env.py` scripts

## Attributes Reference

In addition to all the arguments above, the following attributes are exported.

- `upgraded_revision` - (String) The ID of the new database revision.