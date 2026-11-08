package pngcharts

import (
	"math"
	"sort"

	"astroswarm/worker/internal/server/values"
)

func objects(value any) []map[string]any {
	list, _ := values.List(value)
	output := []map[string]any{}
	for _, item := range list {
		if object, isObject := values.Object(item); isObject {
			output = append(output, object)
		}
	}
	return output
}

func numberOr(object map[string]any, key string, fallback float64) float64 {
	value, found := object[key]
	if !found {
		return fallback
	}
	number, isNumber := values.Number(value)
	if !isNumber {
		return fallback
	}
	return number
}

func numbersOf(value any) []float64 {
	list, _ := values.List(value)
	output := make([]float64, len(list))
	for index, item := range list {
		number, isNumber := values.Number(item)
		if !isNumber {
			number = math.NaN()
		}
		output[index] = number
	}
	return output
}

func sortedBy(rows []map[string]any, key string) []map[string]any {
	sorted := append([]map[string]any{}, rows...)
	sort.SliceStable(sorted, func(left, right int) bool {
		return numberOr(sorted[left], key, 0) < numberOr(sorted[right], key, 0)
	})
	return sorted
}

func column(rows []map[string]any, key string, fallback float64) []float64 {
	output := make([]float64, len(rows))
	for index, row := range rows {
		output[index] = numberOr(row, key, fallback)
	}
	return output
}

func sweepRate(point map[string]any, rateKey, timeKey string) float64 {
	if value, found := point[rateKey]; found && value != nil {
		number, _ := values.Number(value)
		return number
	}
	timeValue, found := point[timeKey]
	if !found {
		return 0.0
	}
	seconds, isNumber := values.Number(timeValue)
	if isNumber && seconds >= 0 {
		return 100.0
	}
	return 0.0
}

func sweepRisk(point map[string]any) float64 {
	if value, found := point["risk"]; found && value != nil {
		number, _ := values.Number(value)
		return number
	}
	return 100.0 - sweepRate(point, "capture_rate", "capture_time")
}
