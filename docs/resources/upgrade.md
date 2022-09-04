---
page_title: "upgrade Resource - terraform-provider-alembic"
subcategory: ""
description: |-
  The upgrade resource allows you to apply Alembic upgrades to a specific revision at deployment time. 
---

# Resource `alembic_upgrade`

The order resource allows you to apply Alembic upgrades to your database during
infrastructure deployment. If environment variables are passed, they are marked
as sensitive, since they likely contain database credentials or configuration
collected from infrastructure deployment.

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

## Attributes Reference

In addition to all the arguments above, the following attributes are exported.

- `upgraded_revision` - (String) The ID of the new database revision.
