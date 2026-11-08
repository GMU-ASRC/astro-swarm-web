package evaluations

import (
	"net/http"
	"strconv"
	"strings"

	"astroswarm/worker/internal/server/entries"
	"astroswarm/worker/internal/server/entrycard"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pngcharts"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

func (api *API) chartEntry(r *http.Request) (*entries.Entry, map[string]any, error) {
	ctx := r.Context()
	entry, err := entries.Get(ctx, api.deps.Store.Pool, r.PathValue("eval_id"))
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		return nil, nil, httpx.NotFound("Evaluation not found")
	}
	summary, err := entries.Summary(ctx, api.deps.Store.Pool, entry)
	if err != nil {
		return nil, nil, err
	}
	return entry, summary, nil
}

func chartMeta(entry *entries.Entry) pngcharts.Meta {
	moment := entry.CompletedAt
	if moment == nil {
		moment = entry.CreatedAt
	}
	dateLabel := ""
	if moment != nil {
		dateLabel = pytime.Isoformat(*moment)[:10]
	}
	return pngcharts.Meta{Username: entry.Username, LevelID: entry.Level(), EvalID: entry.ID, DateLabel: dateLabel}
}

func (api *API) sweepRows(r *http.Request, entry *entries.Entry) ([]map[string]any, error) {
	index, err := api.deps.Replays().SweepIndex(r.Context(), owner(entry))
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(index))
	for _, item := range index {
		if row, isObject := values.Object(item); isObject {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (api *API) renderChart(r *http.Request, kind string, entry *entries.Entry, summary map[string]any) ([]byte, error) {
	meta := chartMeta(entry)
	outcomes, _ := values.List(summary["outcomes"])
	switch kind {
	case "line":
		return pngcharts.Line(outcomes, meta)
	case "bar":
		return pngcharts.Bar(outcomes, meta)
	case "sweep", "capture", "risk":
		rows, err := api.sweepRows(r, entry)
		if err != nil {
			return nil, err
		}
		if kind == "sweep" {
			return pngcharts.DetectionRate(rows, meta)
		}
		if kind == "capture" {
			return pngcharts.CaptureRate(rows, meta)
		}
		return pngcharts.Risk(rows, meta)
	case "trial-risk":
		return pngcharts.TrialRisk(summary["trial_destroyed"], summary["trial_resolved"], meta)
	case "attrition":
		return pngcharts.Attrition(listObjects(summary["attrition"]), meta)
	case "sweep-attrition":
		return pngcharts.SweepAttrition(summary["sweep_attrition"], meta)
	case "ring-capture":
		return pngcharts.RingCapture(summary["sweep_progress"], meta)
	case "ring-risk":
		return pngcharts.RingRisk(summary["sweep_progress"], meta)
	case "ring-attrition":
		return pngcharts.RingAttrition(summary["sweep_progress"], meta)
	case "times":
		return pngcharts.Times(summary["detection_times"], summary["capture_times"], meta)
	}
	return nil, httpx.NotFound("Unknown chart")
}

func listObjects(value any) []map[string]any {
	list, _ := values.List(value)
	rows := []map[string]any{}
	for _, item := range list {
		if row, isObject := values.Object(item); isObject {
			rows = append(rows, row)
		}
	}
	return rows
}

func writePNG(w http.ResponseWriter, png []byte, disposition string) error {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(png)))
	if disposition != "" {
		w.Header().Set("Content-Disposition", disposition)
	}
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(png)
	return err
}

func (api *API) chart(w http.ResponseWriter, r *http.Request) error {
	kind, isPNG := strings.CutSuffix(r.PathValue("chart_file"), ".png")
	if !isPNG || kind == "" {
		return httpx.NotFound("")
	}
	entry, summary, err := api.chartEntry(r)
	if err != nil {
		return err
	}
	png, err := api.renderChart(r, kind, entry, summary)
	if err != nil {
		return err
	}
	return writePNG(w, png, `attachment; filename="`+kind+"_"+entry.ID+`.png"`)
}

func (api *API) thumbnail(w http.ResponseWriter, r *http.Request) error {
	entry, summary, err := api.chartEntry(r)
	if err != nil {
		return err
	}
	card := entrycard.Build(entry.Level(), entry.StatusText(), summary)
	png, err := pngcharts.Thumbnail(entry.Username, card)
	if err != nil {
		return err
	}
	return writePNG(w, png, "")
}
