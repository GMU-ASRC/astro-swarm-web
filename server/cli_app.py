import logging

from flask import Flask

from config import Config
from database import db


def create_cli_app():
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
    app = Flask(__name__)
    app.config.from_object(Config)
    db.init_app(app)
    return app
