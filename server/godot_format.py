import struct

TYPE_NIL = 0
TYPE_BOOL = 1
TYPE_INT = 2
TYPE_FLOAT = 3
TYPE_STRING = 4
TYPE_VECTOR2 = 5
TYPE_COLOR = 20
TYPE_DICTIONARY = 27
TYPE_ARRAY = 28

ENCODE_FLAG_64 = 1 << 16

DEFAULT_ARENA_WIDTH = 1280.0
DEFAULT_ARENA_HEIGHT = 720.0


class GodotDecodeError(Exception):
    pass


class _Reader:
    def __init__(self, data, offset=0):
        self.data = data
        self.offset = offset

    def take(self, count):
        end = self.offset + count
        if end > len(self.data):
            raise GodotDecodeError("unexpected end of Godot variant data")
        chunk = self.data[self.offset:end]
        self.offset = end
        return chunk

    def u32(self):
        return struct.unpack_from("<I", self.take(4))[0]

    def i32(self):
        return struct.unpack_from("<i", self.take(4))[0]

    def i64(self):
        return struct.unpack_from("<q", self.take(8))[0]

    def f32(self):
        return struct.unpack_from("<f", self.take(4))[0]

    def f64(self):
        return struct.unpack_from("<d", self.take(8))[0]


def _read_string(reader):
    length = reader.u32()
    raw = reader.take(length)
    padding = (4 - (length % 4)) % 4
    if padding:
        reader.take(padding)
    return raw.decode("utf-8", errors="replace")


def _color_to_hex(r, g, b):
    def channel(value):
        return max(0, min(255, round(value * 255)))

    return "#%02x%02x%02x" % (channel(r), channel(g), channel(b))


def _read_variant(reader):
    header = reader.u32()
    variant_type = header & 0xFFFF
    is_64 = bool(header & ENCODE_FLAG_64)

    if variant_type == TYPE_NIL:
        return None
    if variant_type == TYPE_BOOL:
        return reader.u32() != 0
    if variant_type == TYPE_INT:
        return reader.i64() if is_64 else reader.i32()
    if variant_type == TYPE_FLOAT:
        return reader.f64() if is_64 else reader.f32()
    if variant_type == TYPE_STRING:
        return _read_string(reader)
    if variant_type == TYPE_VECTOR2:
        x = reader.f64() if is_64 else reader.f32()
        y = reader.f64() if is_64 else reader.f32()
        return (x, y)
    if variant_type == TYPE_COLOR:
        r = reader.f32()
        g = reader.f32()
        b = reader.f32()
        reader.f32()
        return _color_to_hex(r, g, b)
    if variant_type == TYPE_DICTIONARY:
        count = reader.u32() & 0x7FFFFFFF
        result = {}
        for _ in range(count):
            key = _read_variant(reader)
            result[key] = _read_variant(reader)
        return result
    if variant_type == TYPE_ARRAY:
        count = reader.u32() & 0x7FFFFFFF
        return [_read_variant(reader) for _ in range(count)]

    raise GodotDecodeError(f"unsupported Godot variant type: {variant_type}")


def parse_stored_var(data):
    if len(data) < 8:
        raise GodotDecodeError("file too small to be a Godot store_var payload")
    return _read_variant(_Reader(data, 4))


def _species_from_setup(setup):
    robot_types = setup.get("robot_types", [])
    if not isinstance(robot_types, list):
        return []

    species = []
    for entry in robot_types:
        if not isinstance(entry, dict):
            continue
        color = entry.get("color")
        species.append({
            "id": str(entry.get("id", "")),
            "name": str(entry.get("name", "")),
            "color": color if isinstance(color, str) else "#ffffff",
        })
    return species


def _arena_dimensions(setup):
    settings = setup.get("settings", {})
    if not isinstance(settings, dict):
        return DEFAULT_ARENA_WIDTH, DEFAULT_ARENA_HEIGHT
    width = settings.get("arena_width", DEFAULT_ARENA_WIDTH)
    height = settings.get("arena_height", DEFAULT_ARENA_HEIGHT)
    return float(width), float(height)


def extract_metadata(data):
    top = parse_stored_var(data)
    if not isinstance(top, dict):
        raise GodotDecodeError("expected a Dictionary at the top level")

    if "setup" in top and "frames" in top:
        setup = top.get("setup", {})
        frames = top.get("frames", [])
    else:
        setup = top
        frames = []

    if not isinstance(setup, dict):
        setup = {}

    placements = setup.get("placements", [])
    width, height = _arena_dimensions(setup)

    return {
        "species": _species_from_setup(setup),
        "robot_count": len(placements) if isinstance(placements, list) else 0,
        "frame_count": len(frames) if isinstance(frames, list) else 0,
        "arena_width": width,
        "arena_height": height,
    }
