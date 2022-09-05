provider "alembic" {
  project_root = "${path.module}/../" // directory where alembic config is found
  config       = "alembic.ini"        // name of alembic config file (default: alembic.ini)
  section      = "alembic"            // section within config where alembic config is specified (default: alembic)

  // The command used to invoke alembic, which defaults to just
  // ["alembic"].
  alembic = ["poetry", "run", "alembic"]

  // Extra values passed through the -x alembic argument
  // extra = {
  //   provider_extra = "something cool"
  // }
}

locals {
  // Used below for database connections. This could come other another resource(s)
  // which created the database instance, database, and user.
  database_connection_string = "postgresql+psycopg2://user:password@127.0.0.1:5432/databasename"
}

# Upgrade a database
resource "alembic_upgrade" "db-upgrade" {
  revision = "head"          // The revision ID or special "head" value
  tag      = "my-custom-tag" // Custom tag passed with the --tag option

  // Environment variables passed to the alembic command
  environment = {
    DATABASE_URL = locals.database_connection_string
  }

  // These are passed **along with** any extras defined at the provider
  // level using the -x argument to Alembic.
  // extra = {
  //   something = "special"
  // }

  // You can override the alembic command on a per-resource basis
  // alembic = ["custom", "alembic", "command"]

  // If you need a proxy like cloudsql or an SSH port forward for connecting,
  // you can do that here.
  // proxy_command = ["cloud_sql_proxy", "-instances=..."]

  // You can also set a sleep duration for after starting the proxy.
  // This is set to 5s by default, and will happen whenever a connection
  // needs to be made to the database.
  // proxy_sleep = "30s"
}

# Stamp a database; this does not perform and upgrade and only
# marks the database as if it had been upgrade (see 'alembic stamp --help')
resource "alembic_stamp" "db-stamp" {
  revision = "head"          // The revision ID or special "head" value
  tag      = "my-custom-tag" // Custom tag passed with the --tag option

  // Environment variables passed to the alembic command
  environment = {
    DATABASE_URL = locals.database_connection_string
  }

  // These are passed **along with** any extras defined at the provider
  // level using the -x argument to Alembic.
  // extra = {
  //   something = "special"
  // }

  // You can override the alembic command on a per-resource basis
  // alembic = ["custom", "alembic", "command"]

  // If you need a proxy like cloudsql or an SSH port forward for connecting,
  // you can do that here.
  // proxy_command = ["cloud_sql_proxy", "-instances=..."]

  // You can also set a sleep duration for after starting the proxy.
  // This is set to 5s by default, and will happen whenever a connection
  // needs to be made to the database.
  // proxy_sleep = "30s"
}
