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
