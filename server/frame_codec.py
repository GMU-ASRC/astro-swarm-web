import base64
import json
import zlib

FORMAT_DELTA = "delta"
FORMAT_PLAIN = "plain"


def inflate(data):
    try:
        return zlib.decompress(data)
    except zlib.error:
        return zlib.decompress(data, -zlib.MAX_WBITS)


def decode_delta(compressed):
    delta_encoded = json.loads(inflate(compressed))
    if not delta_encoded:
        return []
    frames = [delta_encoded[0]]
    for index in range(1, len(delta_encoded)):
        previous = frames[index - 1]
        delta = delta_encoded[index]
        frames.append([previous[position] + delta[position] for position in range(len(delta))])
    return frames


def decode_plain(compressed):
    return json.loads(inflate(compressed))


def encode_plain(frames):
    return zlib.compress(json.dumps(frames, separators=(",", ":")).encode("utf-8"))


def decode(compressed, frames_format):
    if not compressed:
        return []
    if frames_format == FORMAT_PLAIN:
        return decode_plain(compressed)
    return decode_delta(compressed)


def unpack_base64(packed):
    return base64.b64decode(packed)
