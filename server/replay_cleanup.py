from sqlalchemy import or_

import replay_files
from benchmark_models import BenchmarkRun
from benchmark_store import EXTRA_PACKED_KEY, EXTRA_PLAIN_KEY, INLINE_UPLOAD_ID


def _escape_like(value):
    return value.replace("\\", "\\\\").replace("%", "\\%").replace("_", "\\_")


def _referenced(evaluation_id, upload_id):
    pattern = _escape_like("%s/%s/" % (evaluation_id, upload_id)) + "%"
    if BenchmarkRun.query.filter(BenchmarkRun.frames_key.like(pattern, escape="\\")).first() is not None:
        return True
    if upload_id != INLINE_UPLOAD_ID:
        return False
    extra_keys = [BenchmarkRun.details[key].astext for key in (EXTRA_PACKED_KEY, EXTRA_PLAIN_KEY)]
    return (
        BenchmarkRun.query.filter(or_(*[extra_key.like(pattern, escape="\\") for extra_key in extra_keys])).first()
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
        if upload_id != keep_upload_id and not _referenced(evaluation_id, upload_id)
    ]


def remove_uploads(uploads):
    for evaluation_id, upload_id in uploads:
        replay_files.remove_upload(evaluation_id, upload_id)
