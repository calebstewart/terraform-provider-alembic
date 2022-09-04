---
page_title: "Provider: Alembic"
subcategory: ""
description: |-
  Terraform provider for automatic database schema upgrades with [Alembic](https://alembic.sqlalchemy.org/en/latest/).
---

# Alembic Provider

This provider allows you to easily embed your Alembic database migration execution
into your terraform deployment scripts.

## Example Usage

```terraform
provider "hashicups" {
  project_root = "${path.module}/../"
  alembic = ["poetry", "run", "alembic"]
}
```

## Schema

### Required

- **project_root** (String) Path to the root of your project where `alembic.ini` can be found.

### Optional

- **alembic** (List[String], Optional) The command used to invoke Alembic (default: `["alembic"]`)
