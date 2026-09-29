# Migrations

Plain SQL, applied by `scripts/migrate.sh` in **filename order**, one
transaction per file, tracked in `schema_migrations` on the target database
(the tracking table is created by the script itself).

- Name files `000N_short_name.sql` — zero-padded, because lexicographic order
  is apply order; a number is never reused.
- An applied migration is immutable. Fix forward with a new migration
  (docs/spec/architecture.md).
- Apply to the dev database with `scripts/migrate.sh`; the test database is
  migrated by `scripts/testdb.sh`.
