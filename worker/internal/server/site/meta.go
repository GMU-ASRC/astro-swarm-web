package site

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/entrycard"
	"astroswarm/worker/internal/server/values"
)

const siteDescription = "Design your fleet, program its behavior, and send it against the swarm. " +
	"AstroSwarm is a swarm strategy game built in Godot 4."

type pageText struct {
	title       string
	description string
}

var pageMeta = map[string]pageText{
	"":            {"AstroSwarm", siteDescription},
	"previews":    {"Previews — AstroSwarm", "Screenshots and preview footage from AstroSwarm."},
	"downloads":   {"Downloads — AstroSwarm", "Download the latest AstroSwarm builds."},
	"leaderboard": {"Success Rate Leaderboard — AstroSwarm", "Top commanders ranked by average success rate across every AstroSwarm level."},
	"gamemodes":   {"Game Modes — AstroSwarm", "Browse AstroSwarm game modes and the data behind every recorded run."},
	"levels":      {"Levels — AstroSwarm", "Per-level benchmark data for player algorithms in AstroSwarm."},
	"survive":     {"Survive Matches — AstroSwarm", "Two-player Survive match reports with actions-per-minute telemetry."},
	"simulator":   {"Simulator Entries — AstroSwarm", "Swarm simulator runs shared by players, with replays and the logic of every species."},
}

type pageMetaValues struct {
	title, description, kind, image, url string
}

func (site *Site) baseURL(r *http.Request) string {
	if site.deps.Config.PublicBaseURL != "" {
		return strings.TrimRight(site.deps.Config.PublicBaseURL, "/")
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func pathSegments(path string) []string {
	segments := []string{}
	for _, segment := range strings.Split(path, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

func (site *Site) defaultMeta(r *http.Request, path string) pageMetaValues {
	base := site.baseURL(r)
	return pageMetaValues{title: "AstroSwarm", description: siteDescription, kind: "website",
		image: base + "/previews/Pasted%20image.png", url: base + "/" + path}
}

func (site *Site) resolveMeta(ctx context.Context, r *http.Request, path string) (pageMetaValues, error) {
	meta := site.defaultMeta(r, path)
	segments := pathSegments(path)
	first := ""
	if len(segments) > 0 {
		first = segments[0]
	}
	if text, known := pageMeta[first]; known {
		meta.title, meta.description = text.title, text.description
	}
	if len(segments) < 2 {
		return meta, nil
	}
	switch first {
	case "levels":
		return site.entryMeta(ctx, r, segments[1], meta)
	case "simulator":
		return site.simulatorMeta(ctx, segments[1], meta)
	}
	return meta, nil
}

func (site *Site) entryMeta(ctx context.Context, r *http.Request, entryID string, meta pageMetaValues) (pageMetaValues, error) {
	pool := site.deps.Store.Pool
	entry, err := entries.Get(ctx, pool, entryID)
	if err != nil || entry == nil {
		return meta, err
	}
	summary, err := entries.Summary(ctx, pool, entry)
	if err != nil {
		return meta, err
	}
	card := entrycard.Build(entry.Level(), entry.StatusText(), summary)
	meta.title = entry.Username + " — " + card.LevelName + " — AstroSwarm"
	meta.description = card.Description()
	meta.kind = "article"
	meta.image = site.baseURL(r) + "/api/evaluations/" + entry.ID + "/thumbnail.png"
	return meta, nil
}

func (site *Site) simulatorMeta(ctx context.Context, entryID string, meta pageMetaValues) (pageMetaValues, error) {
	var title, username string
	var description *string
	var speciesRaw []byte
	var frameCount *int
	var recordInterval *float64
	err := site.deps.Store.Pool.QueryRow(ctx, `SELECT title, username, description, species, frame_count, record_interval
		FROM simulator_entries WHERE id = $1`, entryID).Scan(&title, &username, &description, &speciesRaw, &frameCount, &recordInterval)
	if err != nil {
		if isNoRows(err) {
			return meta, nil
		}
		return meta, err
	}
	meta.title = title + " by " + username + " — Simulator — AstroSwarm"
	meta.kind = "article"
	if description != nil && *description != "" {
		meta.description = *description
		return meta, nil
	}
	frames, interval := 0, 0.1
	if frameCount != nil {
		frames = *frameCount
	}
	if recordInterval != nil && *recordInterval != 0 {
		interval = *recordInterval
	}
	species, _ := decodeList(speciesRaw)
	duration := values.FloatRepr(values.RoundTo(float64(frames)*interval, 1))
	meta.description = fmt.Sprintf("A %ss swarm simulator run with %d species.", duration, len(species))
	return meta, nil
}
