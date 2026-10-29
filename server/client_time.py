from datetime import datetime, timedelta, timezone

MAX_CLIENT_TIMESTAMP_AGE = timedelta(days=365)
MAX_CLIENT_TIMESTAMP_LENGTH = 40


def parse_created_at(value):
    now = datetime.now(timezone.utc)
    if value is None or value == "":
        return now
    if not isinstance(value, str) or len(value) > MAX_CLIENT_TIMESTAMP_LENGTH:
        raise ValueError("created_at must be an ISO 8601 date string")
    try:
        stamp = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        raise ValueError("created_at must be an ISO 8601 date string")
    if stamp.tzinfo is None:
        stamp = stamp.replace(tzinfo=timezone.utc)
    return max(now - MAX_CLIENT_TIMESTAMP_AGE, min(stamp, now))
