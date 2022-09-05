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
