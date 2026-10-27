from sqlalchemy import Text, cast

import replay_files
from models import PlayerEvaluation


def _referenced_elsewhere(evaluation_id, upload_id):
    marker = "%s/%s/" % (evaluation_id, upload_id)
    return (
        PlayerEvaluation.query.filter(PlayerEvaluation.id != evaluation_id)
        .filter(cast(PlayerEvaluation.replays, Text).contains(marker))
        .first()
        is not None
    )


def unused_uploads(evaluation_id, keep_upload_id=None):
    try:
        upload_ids = replay_files.upload_ids(evaluation_id)
    except replay_files.ReplayFileError:
        return []
    return [
        (evaluation_id, upload_id)
        for upload_id in upload_ids
        if upload_id != keep_upload_id and not _referenced_elsewhere(evaluation_id, upload_id)
    ]


def remove_uploads(uploads):
    for evaluation_id, upload_id in uploads:
        replay_files.remove_upload(evaluation_id, upload_id)
