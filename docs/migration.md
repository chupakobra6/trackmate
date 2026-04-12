# Trackmate Migration Guide

Trackmate should be migrated with a logical PostgreSQL dump, not by copying the Docker volume.

The preferred migration style is a staged cutover:

1. prepare the target machine ahead of time;
2. restore and validate against a prep dump before cutover;
3. stop the source bot only for the final dump and final restore;
4. start polling on the new machine only after the source polling worker is stopped.

In normal day-to-day work, local `.env` plus local Docker is the development environment, while the VPS that runs the long-lived polling worker is the production environment.

## Why this approach

- A logical dump is portable across machines and easy to verify before restore.
- The dump is a single file that can be copied between environments and verified before restore.
- The backup and restore steps are repeatable and do not depend on Docker volume internals.

## Files to transfer

- database dump: `backups/trackmate_<timestamp>.dump`
- checksum file: `backups/trackmate_<timestamp>.dump.sha256` if present
- metadata file: `backups/trackmate_<timestamp>.dump.meta`
- deployment `.env` file transferred separately from the dump

Do not transfer old Docker containers or the `postgres-data` volume.

## Backup commands

### Routine backup on the source machine

This is safe while the bot is running. It produces a consistent PostgreSQL snapshot, but the bot may write new data after the backup finishes.

```sh
make docker-db-backup
```

If you need a custom path, call the script directly:

```sh
sh scripts/backup_docker_db.sh --output backups/prep.dump
```

### Final cutover backup

Use this immediately before switching to the new machine. It stops `api` and `worker`, writes the dump, verifies the archive, and keeps the application stopped so the old instance cannot create new writes after the backup.

```sh
make docker-db-backup-stop
```

Important:

- Do not restart the old `api` or `worker` after the cutover dump unless you intentionally abort the migration.
- Do not run the old and new Telegram polling workers at the same time.

## Prepare the target machine

Before the final switchover, prepare as much as possible on the target machine:

1. Put the repository on the target machine at the correct commit.
2. Put the target `.env` in place.
3. Install Docker and any host-level prerequisites.
4. Optionally restore a prep dump on the target machine and verify the application stack against that dump.
5. Build the Docker images ahead of time if the target machine is small and image builds are expected to be slow.

This keeps the actual cutover short because the final maintenance window only needs the last dump, the final restore, and the service start.

## Restore on the target machine

1. Put the repository on the target machine at the correct commit.
2. Put the target `.env` in place.
3. Copy the dump file into the repo, for example `backups/trackmate_20260411T120000Z.dump`.
4. Restore it:

```sh
make docker-db-restore FILE=backups/trackmate_20260411T120000Z.dump
```

If needed, the underlying script is:

```sh
sh scripts/restore_docker_db.sh backups/trackmate_20260411T120000Z.dump
```

The restore flow:

- verifies the dump archive with `pg_restore --list`;
- starts Docker PostgreSQL;
- stops `api` and `worker`;
- drops and recreates the target database;
- restores the dump;
- runs the `migrate` service;
- starts `api` and `worker`;
- waits for both services to become healthy.

For the final cutover, this is the recommended restore path because it gives you a verified restore plus a health-checked service start.

## Recommended cutover order

1. On the source machine, create a prep dump while the bot is still running.
2. Copy the prep dump, checksum, metadata, and target `.env` to the target machine.
3. On the target machine, restore the prep dump and validate the stack.
4. When ready to switch over, run `make docker-db-backup-stop` on the source machine.
5. Copy the final dump, checksum, and metadata to the target machine.
6. On the target machine, restore with `make docker-db-restore FILE=<dump-file>`.
7. Verify Docker health with `docker compose ps`.
8. Confirm the bot responds correctly on the target machine.

## Validation after cutover

After the final restore:

- check `docker compose ps` and confirm `postgres`, `api`, and `worker` are healthy;
- follow `docker compose logs -f api worker` for startup errors;
- confirm the bot responds as expected;
- keep the old machine stopped until the new machine is verified.

After the production machine is stable, return to the standard production update flow described in `README.md`.
