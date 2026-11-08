package apicompare

import (
	"encoding/json"
	"fmt"
	"net/url"
)

type Kind int

const (
	KindJSON Kind = iota
	KindZip
	KindBinary
	KindHTML
)

type Route struct {
	Path     string
	Admin    bool
	Kind     Kind
	Unstable string
}

type Discovery struct {
	client       *Client
	base         string
	perLevel     int
	perList      int
	routes       []Route
	seen         map[string]bool
	evaluationID string
}

var chartKinds = []string{"line", "bar", "sweep", "capture", "risk", "trial-risk", "attrition", "sweep-attrition",
	"ring-capture", "ring-risk", "ring-attrition", "times"}

func (discovery *Discovery) add(route Route) {
	if discovery.seen[route.Path] {
		return
	}
	discovery.seen[route.Path] = true
	discovery.routes = append(discovery.routes, route)
}

func (discovery *Discovery) json(path string) any {
	response := discovery.client.Get(discovery.base+path, false)
	if response.Err != nil || response.Status != 200 {
		return nil
	}
	value, err := decodeJSON(response.Body)
	if err != nil {
		return nil
	}
	return value
}

func list(value any) []map[string]any {
	items, _ := value.([]any)
	if object, isObject := value.(map[string]any); isObject {
		items, _ = object["items"].([]any)
		if items == nil {
			items, _ = object["runs"].([]any)
		}
	}
	output := []map[string]any{}
	for _, item := range items {
		if object, isObject := item.(map[string]any); isObject {
			output = append(output, object)
		}
	}
	return output
}

func text(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	}
	return ""
}

func Discover(client *Client, flaskBase string, perLevel int, perList int) []Route {
	discovery := &Discovery{client: client, base: flaskBase, perLevel: perLevel, perList: perList, seen: map[string]bool{}}
	discovery.staticRoutes()
	discovery.evaluations()
	discovery.players()
	discovery.otherContent()
	if discovery.evaluationID != "" {
		discovery.add(Route{Path: "/levels/" + discovery.evaluationID, Kind: KindHTML})
	}
	return discovery.routes
}

func (discovery *Discovery) staticRoutes() {
	for _, path := range []string{
		"/health", "/api/evaluations", "/api/evaluations?exclude_cancelled=1", "/api/evaluations/players",
		"/api/evaluations/baseline", "/api/evaluations/settings", "/api/leaderboard", "/api/survive/matches",
		"/api/survive/matches?limit=5", "/api/training/leaderboard", "/api/training/leaderboard?limit=3",
		"/api/simulator/entries", "/api/simulator/entries?limit=5", "/api/runs", "/api/runs?page=2&page_size=5",
		"/api/evaluations/does-not-exist", "/api/evaluations/players/does-not-exist", "/api/does-not-exist",
		"/api/survive/matches/does-not-exist", "/api/simulator/entries/does-not-exist", "/api/runs/does-not-exist",
	} {
		discovery.add(Route{Path: path})
	}
	for level := 1; level <= 9; level++ {
		levelID := fmt.Sprintf("farp%d", level)
		discovery.add(Route{Path: "/api/evaluations?level_id=" + levelID})
		discovery.add(Route{Path: "/api/evaluations/level-leaderboard?level_id=" + levelID})
		discovery.add(Route{Path: "/api/evaluations/level-sweep?level_id=" + levelID})
		discovery.add(Route{Path: "/api/evaluations/best?level_id=" + levelID, Unstable: "picks at random between tied leaders"})
	}
	discovery.add(Route{Path: "/api/version", Unstable: "asks GitHub"})
	for _, path := range []string{"/api/admin/me", "/api/admin/users", "/api/admin/export/info"} {
		discovery.add(Route{Path: path, Admin: true})
	}
	for _, path := range []string{"/", "/leaderboard", "/gamemodes/levels", "/no-such-page", "/.env", "/robots.txt"} {
		discovery.add(Route{Path: path, Kind: KindHTML})
	}
}

func (discovery *Discovery) evaluations() {
	perLevel := map[string]int{}
	for _, entry := range list(discovery.json("/api/evaluations")) {
		levelID := text(entry["level_id"])
		if perLevel[levelID] >= discovery.perLevel {
			continue
		}
		perLevel[levelID]++
		discovery.evaluationRoutes(text(entry["id"]))
	}
}

func (discovery *Discovery) evaluationRoutes(entryID string) {
	if entryID == "" {
		return
	}
	if discovery.evaluationID == "" {
		discovery.evaluationID = entryID
	}
	base := "/api/evaluations/" + url.PathEscape(entryID)
	discovery.add(Route{Path: base})
	discovery.add(Route{Path: base + "/export", Kind: KindZip})
	discovery.add(Route{Path: base + "/thumbnail.png", Kind: KindBinary})
	for _, chart := range chartKinds {
		discovery.add(Route{Path: base + "/chart/" + chart + ".png", Kind: KindBinary})
	}
	discovery.add(Route{Path: base + "/replays"})
	runs := list(discovery.json(base + "/replays"))
	for index, run := range runs {
		if index == 0 || index == len(runs)-1 {
			discovery.add(Route{Path: base + "/replay/" + text(run["trial"])})
		}
	}
	discovery.add(Route{Path: base + "/replay/999999"})
	discovery.add(Route{Path: base + "/sweep-replays"})
	sweeps := list(discovery.json(base + "/sweep-replays"))
	if len(sweeps) == 0 {
		return
	}
	n := text(sweeps[0]["n"])
	discovery.add(Route{Path: base + "/sweep-replay/" + n})
	discovery.add(Route{Path: base + "/sweep-replay/" + n + "/trials"})
	trials := list(discovery.json(base + "/sweep-replay/" + n + "/trials"))
	if len(trials) > 0 {
		discovery.add(Route{Path: base + "/sweep-replay/" + n + "/trial/" + text(trials[0]["trial"])})
	}
}

func (discovery *Discovery) players() {
	for index, player := range list(discovery.json("/api/evaluations/players")) {
		if index >= discovery.perList {
			break
		}
		discovery.add(Route{Path: "/api/evaluations/players/" + url.PathEscape(text(player["player_id"]))})
	}
}

func (discovery *Discovery) otherContent() {
	type source struct {
		listPath string
		idKey    string
		routes   []string
		kinds    []Kind
	}
	sources := []source{
		{"/api/leaderboard", "id", []string{"/api/leaderboard/%s", "/api/leaderboard/%s/export"}, []Kind{KindJSON, KindZip}},
		{"/api/survive/matches", "id", []string{"/api/survive/matches/%s"}, []Kind{KindJSON}},
		{"/api/training/leaderboard", "player_id", []string{"/api/training/scores/%s"}, []Kind{KindJSON}},
		{"/api/simulator/entries", "id", []string{"/api/simulator/entries/%s", "/api/simulator/entries/%s/replay", "/simulator/%s"},
			[]Kind{KindJSON, KindJSON, KindHTML}},
		{"/api/runs", "id", []string{"/api/runs/%s", "/api/runs/%s/config", "/api/runs/%s/export"}, []Kind{KindJSON, KindJSON, KindZip}},
	}
	for _, current := range sources {
		for index, item := range list(discovery.json(current.listPath)) {
			if index >= discovery.perList {
				break
			}
			itemID := url.PathEscape(text(item[current.idKey]))
			for routeIndex, pattern := range current.routes {
				discovery.add(Route{Path: fmt.Sprintf(pattern, itemID), Kind: current.kinds[routeIndex]})
			}
		}
	}
}
