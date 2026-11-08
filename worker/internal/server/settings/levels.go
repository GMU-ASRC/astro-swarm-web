package settings

import "slices"

type Level struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const (
	CurrentGameVersion  = "v0.0.9"
	AssaultTrials       = 100
	AssaultCleanStreak  = 3
	AssaultSweepMax     = 40
	StreamSweepMax      = 100
	AssaultSweepTrials  = 20
	SiegeEvaders        = 5
	SweepSeedOffset     = 100000
	SweepSeedStride     = 1000000
	SweepMatchOffset    = 500000
	SeedMinimum         = 1
	SeedMaximum         = 2147483647
	SweepSettingMinimum = 1
	SweepSettingMaximum = 1000
)

var Levels = []Level{
	{ID: "farp1", Name: "Level 1 - Defense (Place)"},
	{ID: "farp2", Name: "Level 2 - Defense (Scatter)"},
	{ID: "farp3", Name: "Level 3 - Defense (Waves)"},
	{ID: "farp4", Name: "Level 4 - Defense (Attrition)"},
	{ID: "farp5", Name: "Level 5 - Defense (Siege)"},
}

var PilotLevels = []Level{
	{ID: "farp6", Name: "Level 6 - Swarm (Merge)"},
	{ID: "farp7", Name: "Level 7 - Supply (Allocation)"},
	{ID: "farp8", Name: "Level 8 - Triad (Three planets)"},
	{ID: "farp9", Name: "Pilot - Evasion (Hidden)"},
}

var (
	SupplyLevels    = []string{"farp7", "farp8"}
	SwarmLevels     = []string{"farp6"}
	AssaultLevels   = []string{"farp1", "farp2", "farp3", "farp4", "farp5"}
	AttritionLevels = []string{"farp4", "farp5"}
	StreamLevels    = []string{"farp1", "farp2"}
	levelAliases    = map[string][]string{"farp1": {"farp1", "farp"}}
)

func AllLevels() []Level {
	return append(append([]Level{}, Levels...), PilotLevels...)
}

func LevelIDsFor(levelID string) []string {
	if aliases, found := levelAliases[levelID]; found {
		return aliases
	}
	return []string{levelID}
}

func IsBenchmarkLevel(levelID string) bool {
	return slices.ContainsFunc(Levels, func(level Level) bool { return level.ID == levelID })
}

func IsPilotLevel(levelID string) bool {
	return slices.ContainsFunc(PilotLevels, func(level Level) bool { return level.ID == levelID })
}

func IsSupplyLevel(levelID string) bool {
	return slices.Contains(SupplyLevels, levelID)
}

func IsSwarmLevel(levelID string) bool {
	return slices.Contains(SwarmLevels, levelID)
}

func IsAssaultLevel(levelID string) bool {
	return slices.Contains(AssaultLevels, levelID)
}

func HasAttrition(levelID string) bool {
	return slices.Contains(AttritionLevels, levelID)
}

func AssaultSweepMaxFor(levelID string) int {
	if slices.Contains(StreamLevels, levelID) {
		return StreamSweepMax
	}
	return AssaultSweepMax
}
