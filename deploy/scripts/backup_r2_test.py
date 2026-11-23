import gzip
import json
import os
import pathlib
import stat
import subprocess
import tempfile
import time
import unittest


SCRIPTS_DIR = pathlib.Path(__file__).resolve().parent
REPO_ROOT = SCRIPTS_DIR.parent.parent


def write_exec(path: pathlib.Path, body: str) -> None:
    path.write_text(body, encoding="utf-8")
    path.chmod(0o755)


class BackupScriptsTest(unittest.TestCase):
    def test_pgbackrest_archive_push_fails_closed_without_config(self):
        script = REPO_ROOT / "services/postgres-backup/pgbackrest-archive-push.sh"
        with tempfile.TemporaryDirectory() as tmp:
            res = subprocess.run(
                ["sh", str(script), "pg_wal/000000010000000000000001"],
                env={
                    "PATH": os.environ["PATH"],
                    "PGBACKREST_CONFIG": os.path.join(tmp, "missing.conf"),
                },
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res.returncode, 0)
            self.assertIn("config missing or empty", res.stderr)

    def test_pgbackrest_archive_push_uses_configured_stanza_from_conf(self):
        script = REPO_ROOT / "services/postgres-backup/pgbackrest-archive-push.sh"
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            conf = root / "pgbackrest.conf"
            conf.write_text("[global]\nrepo1-type=s3\n\n[custom-prod]\npg1-path=/data\n", encoding="utf-8")
            mock_bin = root / "bin"
            mock_bin.mkdir()
            pgb_log = root / "pgbackrest.log"
            write_exec(mock_bin / "pgbackrest", f"#!/usr/bin/env sh\necho \"$*\" > '{pgb_log}'\n")
            res = subprocess.run(
                ["sh", str(script), "pg_wal/000000010000000000000001"],
                env={
                    "PATH": f"{mock_bin}:{os.environ['PATH']}",
                    "PGBACKREST_CONFIG": str(conf),
                },
                capture_output=True,
                text=True,
            )
            self.assertEqual(res.returncode, 0, res.stderr)
            self.assertIn("--stanza=custom-prod archive-push pg_wal/000000010000000000000001", pgb_log.read_text(encoding="utf-8"))

    def test_backup_db_pg_succeeds_without_legacy_sql_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "main").mkdir(parents=True)
            (root / "main/config.toml").write_text(
                '[db.default]\nconnection = "postgres"\n'
                'url = "host=postgres user=yourtj dbname=yourtj_main port=5432 sslmode=disable"\n',
                encoding="utf-8",
            )
            mock_bin = root / "bin"
            mock_bin.mkdir()
            write_exec(mock_bin / "docker", "#!/usr/bin/env bash\necho '-- pg_dump ok'\n")
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-db.sh"), "main"],
                env={"PATH": f"{mock_bin}:{os.environ['PATH']}", "YOURTJ_ROOT": str(root), "KEEP": "7"},
                capture_output=True,
                text=True,
            )
            self.assertEqual(res.returncode, 0, res.stderr + res.stdout)
            files = list((root / "snapshots/main").iterdir())
            self.assertEqual(len(files), 1)
            self.assertTrue(files[0].name.endswith(".sql.gz"))

    def test_backup_r2_releases_lock_when_staging_mktemp_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            backup_dir = root / "backup"
            backup_dir.mkdir(parents=True)
            r2_env = backup_dir / "r2.env"
            r2_env.write_text(
                "R2_ACCOUNT_ID=acct\nR2_BUCKET=bkt\nR2_ACCESS_KEY_ID=ak\n"
                "R2_SECRET_ACCESS_KEY=sk\nRESTIC_PASSWORD=pw\n",
                encoding="utf-8",
            )
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-r2.sh")],
                env={
                    "PATH": os.environ["PATH"],
                    "YOURTJ_ROOT": str(root),
                    "R2_ENV_FILE": str(r2_env),
                    "TMPDIR": str(root / "missing-tmpdir"),
                },
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res.returncode, 0)
            self.assertFalse((backup_dir / ".backup-r2.lock").exists())

    def test_verify_pitr_restore_script_invariants(self):
        text = (SCRIPTS_DIR / "verify-pitr-restore.sh").read_text(encoding="utf-8")
        self.assertNotIn("max_connections=20", text)
        self.assertNotIn("--network=none", text)
        self.assertIn("public.anonymous_bindings", text)
        self.assertIn("owner_id", text)
        self.assertNotIn("anonymous_identity_bindings", text)

    def test_compose_archive_command_defaults_fail_closed(self):
        compose = (REPO_ROOT / "deploy/docker-compose.yaml").read_text(encoding="utf-8")
        self.assertNotIn("PG_ARCHIVE_COMMAND:-/bin/true", compose)
        self.assertIn("PG_ARCHIVE_COMMAND:-/bin/false", compose)

    def test_backup_db_pg_creates_0600_gzip_and_rotates(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "main").mkdir(parents=True)
            (root / "main/config.toml").write_text(
                '[db.default]\nconnection = "postgres"\n'
                'url = "host=postgres user=pgcustom dbname=yourtj_main port=5432 sslmode=disable"\n',
                encoding="utf-8",
            )
            (root / ".env").write_text("POSTGRES_USER=pgcustom\n", encoding="utf-8")
            snap_dir = root / "snapshots/main"
            snap_dir.mkdir(parents=True)
            # Pre-seed 2 legacy uncompressed dumps + 1 compressed dump so KEEP=2 rotates oldest out.
            for idx, name in enumerate([
                "pg-yourtj_main-20261001_010000.sql",
                "pg-yourtj_main-20261002_010000.sql",
                "pg-yourtj_main-20261003_010000.sql.gz",
            ]):
                p = snap_dir / name
                p.write_bytes(b"old")
                os.utime(p, (1700000000 + idx * 10, 1700000000 + idx * 10))

            mock_bin = root / "bin"
            mock_bin.mkdir()
            call_log = root / "docker.log"
            write_exec(
                mock_bin / "docker",
                f"#!/usr/bin/env bash\n"
                f"echo \"$*\" >> '{call_log}'\n"
                f"echo '-- PostgreSQL dump ok'\n",
            )

            env = {
                "PATH": f"{mock_bin}:{os.environ['PATH']}",
                "YOURTJ_ROOT": str(root),
                "KEEP": "2",
            }
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-db.sh"), "main"],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(res.returncode, 0, res.stderr + res.stdout)
            self.assertEqual(stat.S_IMODE(snap_dir.stat().st_mode), 0o700)

            remaining = sorted(p.name for p in snap_dir.iterdir())
            self.assertEqual(len(remaining), 2)
            self.assertIn("pg-yourtj_main-20261003_010000.sql.gz", remaining)
            newest = [p for p in snap_dir.iterdir() if p.name != "pg-yourtj_main-20261003_010000.sql.gz"][0]
            self.assertTrue(newest.name.endswith(".sql.gz"))
            self.assertEqual(stat.S_IMODE(newest.stat().st_mode), 0o600)
            self.assertIn("PostgreSQL dump ok", gzip.decompress(newest.read_bytes()).decode("utf-8"))
            self.assertIn("-U pgcustom -d yourtj_main", call_log.read_text(encoding="utf-8"))

    def test_backup_db_pg_fails_closed_on_empty_dump(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "main").mkdir(parents=True)
            (root / "main/config.toml").write_text(
                '[db.default]\nconnection = "postgres"\nurl = "postgres://u:p@postgres/yourtj_main"\n',
                encoding="utf-8",
            )
            mock_bin = root / "bin"
            mock_bin.mkdir()
            write_exec(mock_bin / "docker", "#!/usr/bin/env bash\nexit 1\n")
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-db.sh"), "main"],
                env={"PATH": f"{mock_bin}:{os.environ['PATH']}", "YOURTJ_ROOT": str(root)},
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res.returncode, 0)
            self.assertEqual(list((root / "snapshots/main").iterdir()), [])

    def test_backup_r2_rejects_placeholder_credentials(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            backup_dir = root / "backup"
            backup_dir.mkdir(parents=True)
            r2_env = backup_dir / "r2.env"
            r2_env.write_text((SCRIPTS_DIR.parent / "backup/r2.env.example").read_text(encoding="utf-8"), encoding="utf-8")
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-r2.sh")],
                env={
                    "PATH": os.environ["PATH"],
                    "YOURTJ_ROOT": str(root),
                    "R2_ENV_FILE": str(r2_env),
                },
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(res.returncode, 0)
            self.assertIn("unset or placeholder", res.stderr)

    def test_backup_r2_end_to_end_auto_full_and_restic_upload(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "main/storage/database").mkdir(parents=True)
            (root / "main/storage/oidc").mkdir(parents=True)
            (root / "main/storage/database/file.db").write_bytes(b"sqlite-bytes")
            (root / "main/storage/oidc/signing_key.pem").write_text("PRIVATE", encoding="utf-8")
            (root / "main/config.toml").write_text(
                '[db.default]\nconnection = "postgres"\nurl = "postgres://yourtj:secret@postgres/yourtj_main"\n',
                encoding="utf-8",
            )
            (root / ".env").write_text("POSTGRES_USER=yourtj\n", encoding="utf-8")
            (root / "docker-compose.yaml").write_text("services: {}\n", encoding="utf-8")

            backup_dir = root / "backup"
            backup_dir.mkdir()
            r2_env = backup_dir / "r2.env"
            r2_env.write_text(
                "\n".join([
                    "R2_ACCOUNT_ID=acct123",
                    "R2_BUCKET=yourtj-backups",
                    "R2_ACCESS_KEY_ID=keyid",
                    "R2_SECRET_ACCESS_KEY=secretkey",
                    "R2_RESTIC_PREFIX=restic",
                    "PGBACKREST_CIPHER_PASS=cipherpass",
                    "RESTIC_PASSWORD=resticpass",
                    "PGBACKREST_ENABLED=true",
                    "PGBACKREST_STANZA=main",
                    "LOGICAL_PG_BACKUP=true",
                    "RESTIC_SKIP_PRUNE=false",
                ]) + "\n",
                encoding="utf-8",
            )

            mock_bin = root / "bin"
            mock_bin.mkdir()
            docker_log = root / "docker.log"
            restic_log = root / "restic.log"
            staged_files = root / "staged.log"

            write_exec(
                mock_bin / "docker",
                f"""#!/usr/bin/env bash
echo "$*" >> '{docker_log}'
if [[ "$*" == *"pgbackrest"* && "$*" == *"info --output=json"* ]]; then
  echo '[{{"name":"main","status":{{"code":0}},"backup":[]}}]'
  exit 0
fi
if [[ "$*" == *"pg_dump "* ]]; then
  echo 'PGDMP_MOCK_CONTENT'
  exit 0
fi
if [[ "$*" == *"pg_restore --list"* ]]; then
  cat >/dev/null
  echo '1; 1234 TABLE public users yourtj'
  exit 0
fi
if [[ "$*" == *"pg_dumpall --globals-only"* ]]; then
  echo '-- globals'
  exit 0
fi
exit 0
""",
            )
            write_exec(
                mock_bin / "sqlite3",
                """#!/usr/bin/env python3
import shutil, sys
dst = sys.argv[2].split("'")[1]
shutil.copyfile(sys.argv[1], dst)
""",
            )
            write_exec(
                mock_bin / "restic",
                f"""#!/usr/bin/env bash
echo "REPO=$RESTIC_REPOSITORY ARGS=$*" >> '{restic_log}'
if [ "$1" = "backup" ]; then
  find "$2" -type f | sort > '{staged_files}'
fi
exit 0
""",
            )

            env = {
                "PATH": f"{mock_bin}:{os.environ['PATH']}",
                "YOURTJ_ROOT": str(root),
                "R2_ENV_FILE": str(r2_env),
                "OPENRESTY_BACKUP_DIR": str(root / "nonexistent-openresty"),
            }
            res = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-r2.sh")],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(res.returncode, 0, res.stderr + res.stdout)

            dlog = docker_log.read_text(encoding="utf-8")
            self.assertIn("pgbackrest --stanza=main check", dlog)
            self.assertIn("pgbackrest --stanza=main backup --type=full", dlog)
            self.assertIn("pg_dump --format=custom --compress=6", dlog)
            self.assertIn("pg_restore --list", dlog)

            rlog = restic_log.read_text(encoding="utf-8")
            self.assertIn("REPO=s3:https://acct123.r2.cloudflarestorage.com/yourtj-backups/restic", rlog)
            self.assertIn("ARGS=backup ", rlog)
            self.assertIn("ARGS=forget ", rlog)
            self.assertIn("--group-by host,tags", rlog)

            staged = staged_files.read_text(encoding="utf-8")
            self.assertIn("postgres/yourtj_main.dump", staged)
            self.assertIn("postgres/globals.sql", staged)
            self.assertIn("main-storage/database/file.db", staged)
            self.assertIn("main-storage/oidc/signing_key.pem", staged)
            self.assertIn("config/main-config.toml", staged)
            self.assertIn("config/dot-env", staged)
            self.assertFalse((root / "backup/.backup-r2.lock").exists())

            # Re-run with RESTIC_SKIP_PRUNE=true and verify forget/prune is skipped
            restic_log.write_text("", encoding="utf-8")
            r2_env.write_text(r2_env.read_text(encoding="utf-8").replace("RESTIC_SKIP_PRUNE=false", "RESTIC_SKIP_PRUNE=true"), encoding="utf-8")
            res2 = subprocess.run(
                ["bash", str(SCRIPTS_DIR / "backup-r2.sh"), "--skip-pgbackrest"],
                env=env,
                capture_output=True,
                text=True,
            )
            self.assertEqual(res2.returncode, 0, res2.stderr + res2.stdout)
            rlog2 = restic_log.read_text(encoding="utf-8")
            self.assertIn("ARGS=backup ", rlog2)
            self.assertNotIn("ARGS=forget ", rlog2)

    def test_check_backup_health_ok_and_failure_detection(self):
        now = int(time.time())
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            (root / "backup").mkdir()
            r2_env = root / "backup/r2.env"
            r2_env.write_text(
                "\n".join([
                    "R2_ACCOUNT_ID=acct123",
                    "R2_BUCKET=yourtj-backups",
                    "R2_ACCESS_KEY_ID=keyid",
                    "R2_SECRET_ACCESS_KEY=secretkey",
                    "RESTIC_PASSWORD=resticpass",
                    "PGBACKREST_ENABLED=true",
                    "MAX_WAL_ARCHIVE_AGE_MIN=45",
                    "MAX_BACKUP_AGE_HOURS=36",
                    "MAX_DISK_USAGE_PCT=99",
                    "MAX_PG_WAL_MB=1024",
                ]) + "\n",
                encoding="utf-8",
            )
            mock_bin = root / "bin"
            mock_bin.mkdir()

            def run_health(arch_row: str, pgb_stop: int, wal_kb: int = 102400):
                pgb_json = json.dumps([
                    {
                        "name": "main",
                        "status": {"code": 0, "message": "ok"},
                        "backup": [
                            {"label": "20261123-033000F", "type": "full", "error": False, "timestamp": {"stop": pgb_stop}}
                        ],
                    }
                ])
                write_exec(
                    mock_bin / "docker",
                    f"""#!/usr/bin/env bash
if [[ "$*" == *"du -sk"* ]]; then
  echo "{wal_kb}\t/var/lib/postgresql/data/pg_wal"
  exit 0
fi
if [[ "$*" == *"pg_stat_archiver"* ]]; then
  echo '{arch_row}'
  exit 0
fi
if [[ "$*" == *"pgbackrest"* ]]; then
  echo '{pgb_json}'
  exit 0
fi
exit 0
""",
                )
                write_exec(
                    mock_bin / "restic",
                    """#!/usr/bin/env bash
python3 - <<'PY'
import datetime, json
now_iso = datetime.datetime.now(datetime.timezone.utc).isoformat().replace("+00:00", "Z")
print(json.dumps([{"short_id": "abc12345", "time": now_iso}]))
PY
""",
                )
                return subprocess.run(
                    ["bash", str(SCRIPTS_DIR / "check-backup-health.sh")],
                    env={
                        "PATH": f"{mock_bin}:{os.environ['PATH']}",
                        "YOURTJ_ROOT": str(root),
                        "R2_ENV_FILE": str(r2_env),
                    },
                    capture_output=True,
                    text=True,
                )

            # Healthy case
            ok_cmd = "/usr/local/bin/pgbackrest-archive-push.sh %p"
            ok_res = run_health(f"on|100|0|{now - 300}|0|{now}|{ok_cmd}", now - 3600)
            self.assertEqual(ok_res.returncode, 0, ok_res.stderr + ok_res.stdout)
            self.assertIn("[ok] WAL archiving healthy", ok_res.stdout)
            self.assertIn("[ok] latest pgbackrest backup", ok_res.stdout)
            self.assertIn("[ok] latest restic snapshot", ok_res.stdout)

            # Failing archive_mode=off
            off_res = run_health(f"off|0|0|0|0|{now}|{ok_cmd}", now - 3600)
            self.assertNotEqual(off_res.returncode, 0)
            self.assertIn("archive_mode is 'off'", off_res.stderr)

            # Failing active archiver error (last_failed >= last_archived)
            err_res = run_health(f"on|100|2|{now - 600}|{now - 60}|{now}|{ok_cmd}", now - 3600)
            self.assertNotEqual(err_res.returncode, 0)
            self.assertIn("WAL archiver failing", err_res.stderr)

            # Failing stale pgBackRest backup (> 36h)
            stale_res = run_health(f"on|100|0|{now - 300}|0|{now}|{ok_cmd}", now - 40 * 3600)
            self.assertNotEqual(stale_res.returncode, 0)
            self.assertIn("limit 36h", stale_res.stderr)

            # Failing archive_command=/bin/true even when archiver reports healthy counts
            noop_res = run_health(f"on|100|0|{now - 300}|0|{now}|/bin/true", now - 3600)
            self.assertNotEqual(noop_res.returncode, 0)
            self.assertIn("archive_command", noop_res.stderr)

            # Failing pg_wal backlog (> 1024 MB)
            wal_res = run_health(f"on|100|0|{now - 300}|0|{now}|{ok_cmd}", now - 3600, wal_kb=1500 * 1024)
            self.assertNotEqual(wal_res.returncode, 0)
            self.assertIn("pg_wal size", wal_res.stderr)


if __name__ == "__main__":
    unittest.main()
