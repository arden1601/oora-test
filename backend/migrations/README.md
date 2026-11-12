# Migrations

SQL migrations and seed data, numbered in order (`0001_init.sql`, `0002_seed.sql`, ...).

- The schema changes only through a migration file, never by hand.
- Seed data is fake: invented names, `example.com` emails.
- No migration tool: the files are embedded in the server binary (`embed.go`) and applied in
  name order at startup by `store.Migrate`, each in its own transaction. The table
  `schema_migrations` remembers which files have run.
- A file that has run is never edited. To change something, add a new file.
- The access rules are data in `0002_seed.sql`: the `money` setting on each department and
  the rows of `department_pages`.
