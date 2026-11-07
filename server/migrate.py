import argparse
import logging
import sys

from cli_app import create_cli_app
from database import db
from migrator import runner

logger = logging.getLogger("migrate")


def parse_arguments():
    parser = argparse.ArgumentParser(description="Run pending AstroSwarm database migrations once, in order.")
    parser.add_argument("--dry-run", action="store_true", help="describe pending migrations without writing anything")
    parser.add_argument("--verify-only", action="store_true", help="re-run verification for finished migrations")
    parser.add_argument("--status", action="store_true", help="list every migration and its status")
    return parser.parse_args()


def main():
    arguments = parse_arguments()
    app = create_cli_app()
    with app.app_context():
        engine = db.engine
        try:
            if arguments.status:
                for line in runner.show_status(engine):
                    print(line)
                return 0
            runner.run(engine, logger, dry_run=arguments.dry_run, verify_only=arguments.verify_only)
            return 0
        except runner.MigrationError as error:
            logger.error("%s", error)
            return 1
        finally:
            engine.dispose()


if __name__ == "__main__":
    sys.exit(main())
