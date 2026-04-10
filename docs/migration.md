# Trackmate Migration Runbook

This project should be migrated with a logical Postgres dump, not by copying the Docker volume.

## Files to transfer

- The database dump: `backups/trackmate_<timestamp>.dump`
- The checksum file: `backups/trackmate_<timestamp>.dump.sha256` if present
- The metadata file: `backups/trackmate_<timestamp>.dump.meta`
- The deployment `.env` file transferred separately from the dump

Do not transfer the old Docker containers or the `postgres-data` volume.

## Why this approach

- A logical dump is portable across machines and easy to verify before restore.
- The dump is a single file that can be copied to another laptop and then to the VPS.
- The backup and restore steps are repeatable and do not depend on local Docker volume internals.

## Create a routine backup on the source laptop

This is safe while the bot is running. It produces a consistent Postgres snapshot, but the bot may write new data after the backup finishes.

```sh
sh scripts/backup_docker_db.sh
```

Use `--output` to control the file path:

```sh
sh scripts/backup_docker_db.sh --output backups/prep.dump
```

## Create the final cutover backup

Use this immediately before switching to the new machine. It stops `api` and `worker`, writes the dump, verifies the archive, and keeps the application stopped so the old instance cannot create new writes after the backup.

```sh
sh scripts/backup_docker_db.sh --stop-app
```

Important:

- Do not restart the old `api` or `worker` after the cutover dump unless you intentionally abort the migration.
- Do not run the old and new Telegram polling workers at the same time.

## Restore on the target machine

1. Put the repository on the target machine at the correct commit.
2. Put the target `.env` in place.
3. Copy the dump file into the repo, for example `backups/trackmate_20260411T120000Z.dump`.
4. Restore it:

```sh
sh scripts/restore_docker_db.sh backups/trackmate_20260411T120000Z.dump
```

The restore script:

- verifies the dump archive with `pg_restore --list`
- starts Docker Postgres
- stops `api` and `worker`
- drops and recreates the target database
- restores the dump
- runs the `migrate` service
- starts `api` and `worker`
- waits for both services to become healthy

## Recommended cutover order

1. On the old laptop, run `sh scripts/backup_docker_db.sh --stop-app`.
2. Copy the dump, checksum, metadata, and `.env` to the transfer laptop.
3. Copy those files to the VPS.
4. On the VPS, restore with `sh scripts/restore_docker_db.sh <dump-file>`.
5. Verify Docker health with `docker compose ps`.
6. Confirm the bot responds correctly on the VPS.

## Make targets

From the repository root:

```sh
make docker-db-backup
make docker-db-backup-stop
make docker-db-restore FILE=backups/trackmate_20260411T120000Z.dump
```
