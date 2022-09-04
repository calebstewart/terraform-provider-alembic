provider "alembic" {
  # Directory where alembic.ini is stored
  project_root = "${path.module}/../"
  # Command used to invoke Alembic (default: "alembic")
  alembic = ["poetry", "run", "alembic"]
}

# Upgrade a database
resource "alembic_upgrade" "db-upgrade" {
  revision = "head"
  tag      = "my-custom-tag"

  environment = {
    DATABASE_URL = locals.database_connection_string
  }
}
