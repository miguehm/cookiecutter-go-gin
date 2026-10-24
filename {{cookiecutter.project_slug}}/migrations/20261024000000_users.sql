-- create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "firstname" varchar(255) NOT NULL,
  "lastname" varchar(255) NOT NULL,
  "email" varchar(100) NOT NULL,
  "password" varchar(100) NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "uni_users_email" UNIQUE ("email")
);
-- create index "idx_users_deleted_at" to table: "users"
CREATE INDEX "idx_users_deleted_at" ON "users" ("deleted_at");
