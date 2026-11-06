# Dev database Atlas uses to compute schema diffs.
# By default Atlas starts its own Postgres in Docker; to use a local
# database, override it from the Makefile or the CLI with:
#
#   atlas migrate diff --env gorm --var dev_url="postgres://dev:dev@127.0.0.1:5432/atlas?sslmode=disable"
variable "dev_url" {
  type    = string
  default = "docker://postgres/16/dev?search_path=public"
}

data "external_schema" "gorm" {
  program = [
    "go",
    "-C",
    "tools",
    "run",
    "./cmd/atlas_loader",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = var.dev_url
  migration {
    dir = "file://migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
