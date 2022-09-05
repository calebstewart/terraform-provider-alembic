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
