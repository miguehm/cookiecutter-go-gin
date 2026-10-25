# Dev-database que Atlas usa para calcular diffs de esquema.
# Por defecto Atlas levanta su propio Postgres en Docker; para usar una base
# local se sobrescribe desde el Makefile o la CLI con:
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
