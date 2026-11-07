import argparse
import json
import logging
import os
import shutil
import subprocess
import sys
from datetime import datetime, timezone

from config import Config
from migrator.backups import MANIFEST_NAME

logger = logging.getLogger("backup")

DATABASE_DUMP_NAME = "database.dump"
UPLOADS_FOLDER_NAME = "uploads"


def database_url():
    url = Config.SQLALCHEMY_DATABASE_URI
    for driver in ("postgresql+psycopg2://", "postgresql+psycopg://"):
        if url.startswith(driver):
            return "postgresql://" + url[len(driver):]
    return url


def dump_database(dump_path):
    logger.info("Dumping database to %s", dump_path)
    subprocess.run(
        ["pg_dump", "--format=custom", "--no-owner", "--no-privileges", "--file", dump_path, "--dbname", database_url()],
        check=True,
    )


def check_database_dump(dump_path):
    listing = subprocess.run(["pg_restore", "--list", dump_path], capture_output=True, text=True)
    if listing.returncode != 0:
        return ["pg_restore could not read the dump: %s" % listing.stderr.strip()[:500]]
    table_data_entries = [line for line in listing.stdout.splitlines() if " TABLE DATA " in line]
    if not table_data_entries:
        return ["the dump has no table data"]
    return []


def copy_uploads(source_root, target_root):
    copied_files = 0
    copied_bytes = 0
    problems = []
    for folder, _, file_names in os.walk(source_root):
        relative_folder = os.path.relpath(folder, source_root)
        target_folder = os.path.normpath(os.path.join(target_root, relative_folder))
        os.makedirs(target_folder, exist_ok=True)
        for file_name in file_names:
            source_path = os.path.join(folder, file_name)
            target_path = os.path.join(target_folder, file_name)
            try:
                shutil.copy2(source_path, target_path)
            except OSError as error:
                problems.append("could not copy %s: %s" % (source_path, error))
                continue
            source_size = os.path.getsize(source_path)
            target_size = os.path.getsize(target_path)
            if source_size != target_size:
                problems.append("size mismatch for %s: %d != %d" % (source_path, source_size, target_size))
            copied_files += 1
            copied_bytes += target_size
    return copied_files, copied_bytes, problems


def write_manifest(folder, manifest):
    temporary_path = os.path.join(folder, MANIFEST_NAME + ".tmp")
    with open(temporary_path, "w", encoding="utf-8") as handle:
        json.dump(manifest, handle, indent=2)
    os.replace(temporary_path, os.path.join(folder, MANIFEST_NAME))


def parse_arguments():
    parser = argparse.ArgumentParser(description="Back up the AstroSwarm database and upload files, then verify the backup.")
    parser.add_argument("--target", default=Config.BACKUP_DIR, help="folder that holds the backups")
    return parser.parse_args()


def main():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
    arguments = parse_arguments()
    created_at = datetime.now(timezone.utc)
    folder = os.path.join(arguments.target, "astroswarm-" + created_at.strftime("%Y%m%d-%H%M%S"))
    os.makedirs(folder, exist_ok=False)

    dump_path = os.path.join(folder, DATABASE_DUMP_NAME)
    dump_database(dump_path)
    problems = check_database_dump(dump_path)

    logger.info("Copying %s to %s", Config.UPLOAD_DIR, folder)
    file_count, total_bytes, copy_problems = copy_uploads(Config.UPLOAD_DIR, os.path.join(folder, UPLOADS_FOLDER_NAME))
    problems.extend(copy_problems)

    manifest = {
        "created_at": created_at.isoformat(),
        "database_dump": DATABASE_DUMP_NAME,
        "database_dump_bytes": os.path.getsize(dump_path),
        "uploads_folder": UPLOADS_FOLDER_NAME,
        "upload_file_count": file_count,
        "upload_total_bytes": total_bytes,
        "verified": not problems,
        "problems": problems[:200],
    }
    write_manifest(folder, manifest)

    if problems:
        for problem in problems[:50]:
            logger.error("%s", problem)
        logger.error("Backup at %s is NOT verified (%d problem(s))", folder, len(problems))
        return 1
    logger.info("Backup verified at %s: %d file(s), %d byte(s)", folder, file_count, total_bytes)
    return 0


if __name__ == "__main__":
    sys.exit(main())
