package validate

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"astroswarm/worker/internal/server/values"
)

const (
	maxSimulatorTitle       = 80
	maxSimulatorDescription = 400
	maxSpecies              = 32
	maxVariables            = 64
	maxObstacles            = 500
	maxSpawnZones           = 64
	maxSimulatorFrames      = 6000
	maxRobotSamples         = 250000
	maxArenaSize            = 20000.0
	minRecordInterval       = 0.01
	maxRecordInterval       = 1.0
	valuesPerRobot          = 4
)

var (
	speciesColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	variableTypes       = []string{"int", "string"}
	speciesConfigKeys   = []string{"speed", "turn_rate", "vision", "fov", "size"}
)

type SimulatorSubmission struct {
	PlayerID       string
	Username       string
	Title          string
	Description    string
	GameVersion    string
	ArenaWidth     float64
	ArenaHeight    float64
	Species        []any
	Behaviors      map[string]any
	ArenaProgram   []any
	Variables      []any
	Obstacles      []any
	SpawnZones     []any
	PlacementCount int64
	PeakRobotCount int
	FrameCount     int
	RecordInterval float64
	Replay         SimulatorReplay
	CreatedAt      time.Time
}

type SimulatorReplay struct {
	Interval any     `json:"interval"`
	Robots   [][]any `json:"robots"`
	Frames   [][]any `json:"frames"`
}

func ParseSimulatorSubmission(data map[string]any) (SimulatorSubmission, error) {
	fields, err := NewFields("SimulatorEntrySubmit", data,
		[]string{"player_id", "username", "title"},
		[]string{"description", "game_version", "setup", "replay", "created_at"})
	if err != nil {
		return SimulatorSubmission{}, err
	}
	submission := SimulatorSubmission{}
	if submission.CreatedAt, err = ParseCreatedAt(fields.Value("created_at", "")); err != nil {
		return submission, err
	}
	if submission.PlayerID, err = requirePlayerID(fields.Value("player_id", nil)); err != nil {
		return submission, err
	}
	if submission.Username, err = requireUsername(fields.Value("username", nil)); err != nil {
		return submission, err
	}
	if submission.Title, err = simulatorText(fields.Value("title", nil), "title", maxSimulatorTitle); err != nil {
		return submission, err
	}
	if submission.Title == "" {
		return submission, fmt.Errorf("title is required")
	}
	if submission.Description, err = simulatorText(fields.Value("description", ""), "description", maxSimulatorDescription); err != nil {
		return submission, err
	}
	if submission.GameVersion, err = simulatorText(fields.Value("game_version", ""), "game_version", 20); err != nil {
		return submission, err
	}
	setup, isObject := fields.Value("setup", map[string]any{}).(map[string]any)
	if !isObject {
		return submission, fmt.Errorf("setup must be an object")
	}
	replay, isObject := fields.Value("replay", map[string]any{}).(map[string]any)
	if !isObject {
		return submission, fmt.Errorf("replay must be an object")
	}
	if err := submission.cleanSetup(setup); err != nil {
		return submission, err
	}
	return submission, submission.cleanReplay(replay)
}

func (submission *SimulatorSubmission) cleanSetup(setup map[string]any) error {
	var err error
	if submission.ArenaWidth, submission.ArenaHeight, err = cleanArena(setup["arena"]); err != nil {
		return err
	}
	if submission.Species, err = cleanSpecies(setup["species"]); err != nil {
		return err
	}
	speciesIDs := submission.speciesIDs()
	budget := &blockBudget{remaining: maxBlocksPerEntry}
	behaviors, isObject := setup["behaviors"].(map[string]any)
	if !isObject {
		return fmt.Errorf("setup.behaviors must be an object keyed by species id")
	}
	submission.Behaviors = map[string]any{}
	for speciesID, scripts := range behaviors {
		if !speciesIDs[speciesID] {
			continue
		}
		if submission.Behaviors[speciesID], err = cleanScripts(scripts, budget); err != nil {
			return err
		}
	}
	if submission.ArenaProgram, err = cleanScripts(objectValue(setup, "arena_program", []any{}), budget); err != nil {
		return err
	}
	if submission.Variables, err = cleanVariables(objectValue(setup, "variables", []any{})); err != nil {
		return err
	}
	if submission.Obstacles, err = cleanObstacles(objectValue(setup, "obstacles", []any{})); err != nil {
		return err
	}
	if submission.SpawnZones, err = cleanSpawnZones(objectValue(setup, "spawn_zones", []any{}), speciesIDs); err != nil {
		return err
	}
	placementCount, err := integerNumber(objectValue(setup, "placement_count", int64(0)), "placement_count")
	if err != nil {
		return err
	}
	submission.PlacementCount = max(0, placementCount)
	return nil
}

func (submission *SimulatorSubmission) speciesIDs() map[string]bool {
	ids := map[string]bool{}
	for _, entry := range submission.Species {
		ids[entry.(map[string]any)["id"].(string)] = true
	}
	return ids
}

func cleanArena(value any) (float64, float64, error) {
	arena, isList := value.([]any)
	if !isList || len(arena) != 2 {
		return 0, 0, fmt.Errorf("setup.arena must be [width, height]")
	}
	width, err := simulatorNumber(arena[0], "arena width")
	if err != nil {
		return 0, 0, err
	}
	height, err := simulatorNumber(arena[1], "arena height")
	if err != nil {
		return 0, 0, err
	}
	widthValue, heightValue := numberFloat(width), numberFloat(height)
	if widthValue < 1.0 || widthValue > maxArenaSize || heightValue < 1.0 || heightValue > maxArenaSize {
		return 0, 0, fmt.Errorf("arena size must be between 1 and %d pixels", int(maxArenaSize))
	}
	return widthValue, heightValue, nil
}

func cleanSpecies(value any) ([]any, error) {
	species, isList := value.([]any)
	if !isList || len(species) < 1 || len(species) > maxSpecies {
		return nil, fmt.Errorf("setup.species must hold between 1 and %d entries", maxSpecies)
	}
	cleaned := []any{}
	seen := map[string]bool{}
	for _, item := range species {
		entry, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("each species must be an object")
		}
		speciesID, err := simulatorText(entry["id"], "species id", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		if speciesID == "" || seen[speciesID] {
			return nil, fmt.Errorf("species ids must be present and unique")
		}
		seen[speciesID] = true
		color := values.ToText(objectValue(entry, "color", ""))
		if !speciesColorPattern.MatchString(color) {
			return nil, fmt.Errorf("species color must be a #rrggbb hex string")
		}
		config, _ := entry["config"].(map[string]any)
		cleanedConfig := map[string]any{}
		for _, key := range speciesConfigKeys {
			if setting, found := config[key]; found {
				if cleanedConfig[key], err = roundedNumber(setting, key, 3); err != nil {
					return nil, err
				}
			}
		}
		name, err := simulatorText(entry["name"], "species name", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = speciesID
		}
		cleaned = append(cleaned, map[string]any{"id": speciesID, "name": name, "color": strings.ToLower(color), "config": cleanedConfig})
	}
	return cleaned, nil
}

func objectList(value any, limit int, message string, itemMessage string) ([]map[string]any, error) {
	list, isList := value.([]any)
	if !isList || len(list) > limit {
		return nil, fmt.Errorf("%s", message)
	}
	objects := make([]map[string]any, 0, len(list))
	for _, item := range list {
		object, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("%s", itemMessage)
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func cleanVariables(value any) ([]any, error) {
	variables, err := objectList(value, maxVariables,
		fmt.Sprintf("setup.variables must be a list of up to %d entries", maxVariables), "each variable must be an object")
	if err != nil {
		return nil, err
	}
	cleaned := []any{}
	for _, entry := range variables {
		name, err := simulatorText(entry["name"], "variable name", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		variableType, _ := objectValue(entry, "type", "int").(string)
		if !containsText(variableTypes, variableType) {
			variableType = "int"
		}
		cleaned = append(cleaned, map[string]any{"name": name, "type": variableType})
	}
	return cleaned, nil
}

func cleanObstacles(value any) ([]any, error) {
	obstacles, err := objectList(value, maxObstacles,
		fmt.Sprintf("setup.obstacles must be a list of up to %d entries", maxObstacles), "each obstacle must be an object")
	if err != nil {
		return nil, err
	}
	cleaned := []any{}
	for _, entry := range obstacles {
		switch entry["type"] {
		case "wall":
			position, err := simulatorPoint(entry["position"])
			if err != nil {
				return nil, err
			}
			size, err := simulatorPoint(entry["size"])
			if err != nil {
				return nil, err
			}
			cleaned = append(cleaned, map[string]any{"type": "wall", "position": position, "size": size})
		case "circle":
			position, err := simulatorPoint(entry["position"])
			if err != nil {
				return nil, err
			}
			radius, err := roundedNumber(entry["radius"], "radius", 1)
			if err != nil {
				return nil, err
			}
			cleaned = append(cleaned, map[string]any{"type": "circle", "position": position, "radius": radius})
		}
	}
	return cleaned, nil
}

func cleanSpawnZones(value any, speciesIDs map[string]bool) ([]any, error) {
	zones, err := objectList(value, maxSpawnZones,
		fmt.Sprintf("setup.spawn_zones must be a list of up to %d entries", maxSpawnZones), "each spawn zone must be an object")
	if err != nil {
		return nil, err
	}
	cleaned := []any{}
	for _, entry := range zones {
		typeID := values.ToText(objectValue(entry, "type_id", ""))
		if !speciesIDs[typeID] {
			typeID = ""
		}
		zoneID, err := integerNumber(objectValue(entry, "id", int64(len(cleaned))), "zone id")
		if err != nil {
			return nil, err
		}
		name, err := simulatorText(entry["name"], "zone name", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = "Zone"
		}
		position, err := simulatorPoint(entry["position"])
		if err != nil {
			return nil, err
		}
		size, err := simulatorPoint(entry["size"])
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, map[string]any{"id": zoneID, "name": name, "position": position, "size": size,
			"type_id": typeID, "enabled": values.Truthy(objectValue(entry, "enabled", true))})
	}
	return cleaned, nil
}
