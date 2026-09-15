import math
import re
from dataclasses import dataclass, field

MAX_TITLE_LENGTH = 80
MAX_DESCRIPTION_LENGTH = 400
MAX_NAME_LENGTH = 40
MAX_TEXT_PARAM_LENGTH = 60
MAX_SPECIES = 32
MAX_SCRIPTS_PER_PROGRAM = 64
MAX_BLOCKS_PER_ENTRY = 4000
MAX_BLOCK_DEPTH = 16
MAX_PARAMS_PER_BLOCK = 12
MAX_VARIABLES = 64
MAX_OBSTACLES = 500
MAX_SPAWN_ZONES = 64
MAX_FRAMES = 6000
MAX_ROBOT_SAMPLES = 250000
MAX_ARENA_SIZE = 20000.0
MIN_RECORD_INTERVAL = 0.01
MAX_RECORD_INTERVAL = 1.0
VALUES_PER_ROBOT = 4
COLOR_PATTERN = re.compile(r"^#[0-9a-fA-F]{6}$")
VARIABLE_TYPES = ("int", "string")
SPECIES_CONFIG_KEYS = ("speed", "turn_rate", "vision", "fov", "size")


@dataclass
class SimulatorEntrySubmit:
    player_id: str
    username: str
    title: str
    description: str = ""
    game_version: str = ""
    setup: dict = field(default_factory=dict)
    replay: dict = field(default_factory=dict)

    def __post_init__(self):
        if not isinstance(self.player_id, str) or len(self.player_id) != 36:
            raise ValueError("player_id must be exactly 36 characters")
        if not isinstance(self.username, str) or not (1 <= len(self.username) <= 30):
            raise ValueError("username must be between 1 and 30 characters")
        self.title = _text(self.title, "title", MAX_TITLE_LENGTH)
        if not self.title:
            raise ValueError("title is required")
        self.description = _text(self.description, "description", MAX_DESCRIPTION_LENGTH)
        self.game_version = _text(self.game_version, "game_version", 20)
        if not isinstance(self.setup, dict):
            raise ValueError("setup must be an object")
        if not isinstance(self.replay, dict):
            raise ValueError("replay must be an object")

        self.arena_width, self.arena_height = self._clean_arena(self.setup.get("arena"))
        self.species = self._clean_species(self.setup.get("species"))
        species_ids = {entry["id"] for entry in self.species}
        block_budget = [MAX_BLOCKS_PER_ENTRY]
        self.behaviors = self._clean_behaviors(self.setup.get("behaviors"), species_ids, block_budget)
        self.arena_program = _clean_scripts(self.setup.get("arena_program", []), block_budget)
        self.variables = self._clean_variables(self.setup.get("variables", []))
        self.obstacles = self._clean_obstacles(self.setup.get("obstacles", []))
        self.spawn_zones = self._clean_spawn_zones(self.setup.get("spawn_zones", []), species_ids)
        self.placement_count = max(0, int(_number(self.setup.get("placement_count", 0), "placement_count")))
        self._clean_replay(species_ids)

    def _clean_arena(self, arena):
        if not isinstance(arena, list) or len(arena) != 2:
            raise ValueError("setup.arena must be [width, height]")
        width = _number(arena[0], "arena width")
        height = _number(arena[1], "arena height")
        if not (1.0 <= width <= MAX_ARENA_SIZE and 1.0 <= height <= MAX_ARENA_SIZE):
            raise ValueError("arena size must be between 1 and %d pixels" % MAX_ARENA_SIZE)
        return width, height

    def _clean_species(self, species):
        if not isinstance(species, list) or not (1 <= len(species) <= MAX_SPECIES):
            raise ValueError("setup.species must hold between 1 and %d entries" % MAX_SPECIES)
        cleaned = []
        seen_ids = set()
        for entry in species:
            if not isinstance(entry, dict):
                raise ValueError("each species must be an object")
            species_id = _text(entry.get("id"), "species id", MAX_NAME_LENGTH)
            if not species_id or species_id in seen_ids:
                raise ValueError("species ids must be present and unique")
            seen_ids.add(species_id)
            color = str(entry.get("color", ""))
            if not COLOR_PATTERN.match(color):
                raise ValueError("species color must be a #rrggbb hex string")
            config = entry.get("config") if isinstance(entry.get("config"), dict) else {}
            cleaned.append({
                "id": species_id,
                "name": _text(entry.get("name"), "species name", MAX_NAME_LENGTH) or species_id,
                "color": color.lower(),
                "config": {key: round(_number(config[key], key), 3) for key in SPECIES_CONFIG_KEYS if key in config},
            })
        return cleaned

    def _clean_behaviors(self, behaviors, species_ids, block_budget):
        if not isinstance(behaviors, dict):
            raise ValueError("setup.behaviors must be an object keyed by species id")
        return {
            species_id: _clean_scripts(scripts, block_budget)
            for species_id, scripts in behaviors.items()
            if species_id in species_ids
        }

    def _clean_variables(self, variables):
        if not isinstance(variables, list) or len(variables) > MAX_VARIABLES:
            raise ValueError("setup.variables must be a list of up to %d entries" % MAX_VARIABLES)
        cleaned = []
        for entry in variables:
            if not isinstance(entry, dict):
                raise ValueError("each variable must be an object")
            variable_type = entry.get("type", "int")
            cleaned.append({
                "name": _text(entry.get("name"), "variable name", MAX_NAME_LENGTH),
                "type": variable_type if variable_type in VARIABLE_TYPES else "int",
            })
        return cleaned

    def _clean_obstacles(self, obstacles):
        if not isinstance(obstacles, list) or len(obstacles) > MAX_OBSTACLES:
            raise ValueError("setup.obstacles must be a list of up to %d entries" % MAX_OBSTACLES)
        cleaned = []
        for entry in obstacles:
            if not isinstance(entry, dict):
                raise ValueError("each obstacle must be an object")
            kind = entry.get("type")
            if kind == "wall":
                cleaned.append({"type": "wall", "position": _point(entry.get("position")), "size": _point(entry.get("size"))})
            elif kind == "circle":
                cleaned.append({"type": "circle", "position": _point(entry.get("position")), "radius": round(_number(entry.get("radius"), "radius"), 1)})
        return cleaned

    def _clean_spawn_zones(self, zones, species_ids):
        if not isinstance(zones, list) or len(zones) > MAX_SPAWN_ZONES:
            raise ValueError("setup.spawn_zones must be a list of up to %d entries" % MAX_SPAWN_ZONES)
        cleaned = []
        for entry in zones:
            if not isinstance(entry, dict):
                raise ValueError("each spawn zone must be an object")
            type_id = str(entry.get("type_id", ""))
            cleaned.append({
                "id": int(_number(entry.get("id", len(cleaned)), "zone id")),
                "name": _text(entry.get("name"), "zone name", MAX_NAME_LENGTH) or "Zone",
                "position": _point(entry.get("position")),
                "size": _point(entry.get("size")),
                "type_id": type_id if type_id in species_ids else "",
                "enabled": bool(entry.get("enabled", True)),
            })
        return cleaned

    def _clean_replay(self, species_ids):
        interval = _number(self.replay.get("interval", 0.1), "replay interval")
        if not (MIN_RECORD_INTERVAL <= interval <= MAX_RECORD_INTERVAL):
            raise ValueError("replay interval must be between %s and %s seconds" % (MIN_RECORD_INTERVAL, MAX_RECORD_INTERVAL))

        robots = self.replay.get("robots", [])
        if not isinstance(robots, list):
            raise ValueError("replay.robots must be a list of [id, species id] pairs")
        robot_species = []
        for pair in robots:
            if not isinstance(pair, list) or len(pair) != 2:
                raise ValueError("replay.robots must be a list of [id, species id] pairs")
            robot_species.append([int(_number(pair[0], "robot id")), str(pair[1]) if str(pair[1]) in species_ids else ""])

        frames = self.replay.get("frames", [])
        if not isinstance(frames, list) or not (1 <= len(frames) <= MAX_FRAMES):
            raise ValueError("replay.frames must hold between 1 and %d frames" % MAX_FRAMES)
        samples = 0
        peak_robots = 0
        cleaned_frames = []
        for frame in frames:
            if not isinstance(frame, list) or len(frame) % VALUES_PER_ROBOT != 0:
                raise ValueError("each replay frame must be a flat list of [id, x, y, rotation] groups")
            robots_in_frame = len(frame) // VALUES_PER_ROBOT
            samples += robots_in_frame
            if samples > MAX_ROBOT_SAMPLES:
                raise ValueError("replay is too large: at most %d robot samples across all frames" % MAX_ROBOT_SAMPLES)
            peak_robots = max(peak_robots, robots_in_frame)
            cleaned_frame = []
            for offset in range(0, len(frame), VALUES_PER_ROBOT):
                cleaned_frame.append(int(_number(frame[offset], "robot id")))
                cleaned_frame.append(round(_number(frame[offset + 1], "x"), 1))
                cleaned_frame.append(round(_number(frame[offset + 2], "y"), 1))
                cleaned_frame.append(round(_number(frame[offset + 3], "rotation"), 1))
            cleaned_frames.append(cleaned_frame)

        self.record_interval = interval
        self.frame_count = len(cleaned_frames)
        self.peak_robot_count = peak_robots
        self.replay = {"interval": interval, "robots": robot_species, "frames": cleaned_frames}


def _clean_scripts(scripts, block_budget):
    if not isinstance(scripts, list) or len(scripts) > MAX_SCRIPTS_PER_PROGRAM:
        raise ValueError("a program must be a list of up to %d scripts" % MAX_SCRIPTS_PER_PROGRAM)
    cleaned = []
    for script in scripts:
        if not isinstance(script, dict):
            raise ValueError("each script must be an object")
        cleaned.append({
            "x": round(_number(script.get("x", 0.0), "script x"), 1),
            "y": round(_number(script.get("y", 0.0), "script y"), 1),
            "blocks": _clean_blocks(script.get("blocks", []), block_budget, 0),
        })
    return cleaned


def _clean_blocks(blocks, block_budget, depth):
    if depth > MAX_BLOCK_DEPTH:
        raise ValueError("blocks are nested more than %d levels deep" % MAX_BLOCK_DEPTH)
    if not isinstance(blocks, list):
        raise ValueError("blocks must be a list")
    cleaned = []
    for block in blocks:
        if not isinstance(block, dict):
            raise ValueError("each block must be an object")
        block_budget[0] -= 1
        if block_budget[0] < 0:
            raise ValueError("an entry may hold at most %d blocks" % MAX_BLOCKS_PER_ENTRY)
        cleaned.append({
            "type": _text(block.get("type"), "block type", MAX_NAME_LENGTH),
            "params": _clean_params(block.get("params", {})),
            "children": _clean_blocks(block.get("children", []), block_budget, depth + 1),
        })
    return cleaned


def _clean_params(params):
    if not isinstance(params, dict) or len(params) > MAX_PARAMS_PER_BLOCK:
        raise ValueError("block params must be an object of up to %d values" % MAX_PARAMS_PER_BLOCK)
    cleaned = {}
    for key, value in params.items():
        name = _text(key, "param name", MAX_NAME_LENGTH)
        if isinstance(value, bool):
            cleaned[name] = value
        elif isinstance(value, (int, float)):
            cleaned[name] = _number(value, name)
        else:
            cleaned[name] = _text(value, name, MAX_TEXT_PARAM_LENGTH)
    return cleaned


def _point(value):
    if not isinstance(value, list) or len(value) != 2:
        raise ValueError("points must be [x, y]")
    return [round(_number(value[0], "x"), 1), round(_number(value[1], "y"), 1)]


def _number(value, label):
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise ValueError("%s must be a number" % label)
    if not math.isfinite(value):
        raise ValueError("%s must be finite" % label)
    return value


def _text(value, label, limit):
    if value is None:
        return ""
    if not isinstance(value, (str, int, float)) or isinstance(value, bool):
        raise ValueError("%s must be text" % label)
    text = str(value).strip()
    if len(text) > limit:
        raise ValueError("%s must be at most %d characters" % (label, limit))
    return text
