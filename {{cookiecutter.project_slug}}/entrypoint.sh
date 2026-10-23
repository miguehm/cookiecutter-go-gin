#!/bin/sh
set -eu

{% if cookiecutter.use_db == "yes" %}
export DATABASE_URL="postgresql://$DB_USER:$DB_PASS@$DB_HOST:$DB_PORT/$DB_NAME"

if [ -f migrations/atlas.sum ]; then
	atlas migrate apply --url "$DATABASE_URL" --dir file://migrations
fi
{% endif %}

exec ./http
