from sqlalchemy import text

from migrator import state


class MigrationContext:
    def __init__(self, engine, connection, migration, logger, dry_run=False):
        self.engine = engine
        self.connection = connection
        self.migration = migration
        self.logger = logger
        self.dry_run = dry_run

    def execute(self, statement, parameters=None):
        result = self.connection.execute(text(statement), parameters or {})
        self.connection.commit()
        return result

    def execute_outside_transaction(self, statement):
        with self.engine.connect().execution_options(isolation_level="AUTOCOMMIT") as autocommit_connection:
            autocommit_connection.execute(text(statement))

    def scalar(self, statement, parameters=None):
        value = self.connection.execute(text(statement), parameters or {}).scalar()
        self.connection.commit()
        return value

    def rows(self, statement, parameters=None):
        result = list(self.connection.execute(text(statement), parameters or {}).mappings())
        self.connection.commit()
        return result

    def last_batch_key(self):
        return state.load_batch_key(self.connection, self.migration.version)

    def save_batch_key(self, batch_key):
        state.save_batch_key(self.connection, self.migration.version, batch_key)

    def log(self, message, *arguments):
        self.logger.info("[%03d %s] " + message, self.migration.version, self.migration.name, *arguments)
