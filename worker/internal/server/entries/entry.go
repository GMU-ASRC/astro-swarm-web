package entries

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"
	"unicode"

	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/store"
)

const CurrentStorageVersion = 2

type Entry struct {
	ID             string
	PlayerID       string
	Username       string
	LevelID        *string
	Algorithm      []byte
	Placements     []byte
	GenomeSchedule []byte
	Status         *string
	Progress       *float64
	Stage          *string
	WorkerID       *string
	Trials         *int
	Error          *string
	BenchmarkID    *string
	JobConfig      []byte
	DoneUnits      *int
	TotalUnits     *int
	LastUpdate     *time.Time
	GameVersion    *string
	DefenderCount  *int
	XPAwarded      *int
	CoinsAwarded   *int
	Collisions     *bool
	CreatedAt      *time.Time
	CompletedAt    *time.Time

	SuccessRate  *float64
	AttackerRate *float64
	DefenderRate *float64
}

const entryColumns = `player_evaluations.id, player_id, username, player_evaluations.level_id, algorithm, placements, genome_schedule,
	status, progress, stage, worker_id, trials, error, benchmark_id, job_config, done_units, total_units, last_update,
	game_version, defender_count, xp_awarded, coins_awarded, collisions, player_evaluations.created_at, completed_at,
	benchmarks.success_rate, benchmarks.attacker_rate, benchmarks.defender_rate`

const entryFrom = ` FROM player_evaluations LEFT JOIN benchmarks ON benchmarks.id = player_evaluations.benchmark_id`

func SelectEntries(condition string) string {
	query := "SELECT " + entryColumns + entryFrom
	if condition != "" {
		query += " WHERE " + condition
	}
	return query
}

type scanner interface {
	Scan(destinations ...any) error
}

func scanEntry(row scanner) (*Entry, error) {
	entry := &Entry{}
	err := row.Scan(&entry.ID, &entry.PlayerID, &entry.Username, &entry.LevelID, &entry.Algorithm, &entry.Placements,
		&entry.GenomeSchedule, &entry.Status, &entry.Progress, &entry.Stage, &entry.WorkerID, &entry.Trials, &entry.Error,
		&entry.BenchmarkID, &entry.JobConfig, &entry.DoneUnits, &entry.TotalUnits, &entry.LastUpdate, &entry.GameVersion,
		&entry.DefenderCount, &entry.XPAwarded, &entry.CoinsAwarded, &entry.Collisions, &entry.CreatedAt, &entry.CompletedAt,
		&entry.SuccessRate, &entry.AttackerRate, &entry.DefenderRate)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func Get(ctx context.Context, db store.Querier, entryID string) (*Entry, error) {
	entry, err := scanEntry(db.QueryRow(ctx, SelectEntries("player_evaluations.id = $1"), entryID))
	if store.IsNoRows(err) {
		return nil, nil
	}
	return entry, err
}

func GetForUpdate(ctx context.Context, db store.Querier, entryID string) (*Entry, error) {
	entry, err := scanEntry(db.QueryRow(ctx, SelectEntries("player_evaluations.id = $1")+" FOR UPDATE OF player_evaluations", entryID))
	if store.IsNoRows(err) {
		return nil, nil
	}
	return entry, err
}

func List(ctx context.Context, db store.Querier, query string, arguments ...any) ([]*Entry, error) {
	rows, err := db.Query(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	listed := []*Entry{}
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		listed = append(listed, entry)
	}
	return listed, rows.Err()
}

func (entry *Entry) Level() string {
	if entry.LevelID == nil || *entry.LevelID == "" {
		return "farp"
	}
	return *entry.LevelID
}

func (entry *Entry) LevelNumber() int {
	levelID := strings.ToLower(entry.Level())
	digits := strings.Builder{}
	for _, character := range levelID {
		if unicode.IsDigit(character) {
			digits.WriteRune(character)
		}
	}
	if digits.Len() == 0 {
		return 1
	}
	number := 0
	for _, character := range digits.String() {
		number = number*10 + int(character-'0')
	}
	return number
}

func (entry *Entry) IsAttackLevel() bool {
	number := entry.LevelNumber()
	return number == 6 || number == 9
}

func (entry *Entry) StatusText() string {
	if entry.Status == nil {
		return ""
	}
	return *entry.Status
}

func decodeAny(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if decoder.Decode(&decoded) != nil {
		return nil
	}
	return decoded
}

func listOrEmpty(raw []byte) any {
	decoded := decodeAny(raw)
	if decoded == nil {
		return []any{}
	}
	if list, isList := decoded.([]any); isList && len(list) == 0 {
		return []any{}
	}
	if object, isObject := decoded.(map[string]any); isObject && len(object) == 0 {
		return []any{}
	}
	return decoded
}

func (entry *Entry) AlgorithmValue() any      { return listOrEmpty(entry.Algorithm) }
func (entry *Entry) PlacementsValue() any     { return listOrEmpty(entry.Placements) }
func (entry *Entry) GenomeScheduleValue() any { return listOrEmpty(entry.GenomeSchedule) }

func (entry *Entry) GenomeScheduleList() []any {
	list, _ := entry.GenomeScheduleValue().([]any)
	return list
}

func (entry *Entry) JobConfigValue() any {
	return decodeAny(entry.JobConfig)
}

func (entry *Entry) GenomeColors() []any {
	colors := []any{}
	for _, stage := range entry.GenomeScheduleList() {
		object, _ := stage.(map[string]any)
		color, isText := object["color"].(string)
		if isText {
			colors = append(colors, "#"+color)
		} else {
			colors = append(colors, nil)
		}
	}
	return colors
}

func stringOr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func intOr(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func floatOr(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func createdAtText(entry *Entry) any {
	if entry.CreatedAt == nil {
		return nil
	}
	return pytime.Isoformat(*entry.CreatedAt)
}

func numberValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	}
	return 0, false
}

func WholeNumber(value any) (int64, bool) {
	number, isNumber := numberValue(value)
	if !isNumber || number != float64(int64(number)) {
		return 0, false
	}
	return int64(number), true
}
