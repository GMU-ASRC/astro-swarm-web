package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"astroswarm/worker/internal/apicompare"
)

func main() {
	flaskBase := flag.String("flask", "http://localhost:5051", "base URL of the Flask server")
	goBase := flag.String("go", "http://localhost:5050", "base URL of the Go server")
	adminKey := flag.String("key", os.Getenv("API_SECRET_KEY"), "admin key or session token for admin routes")
	perLevel := flag.Int("per-level", 8, "entries to check per level")
	perList := flag.Int("per-list", 10, "players, matches, scores, simulator entries and runs to check")
	verbose := flag.Bool("verbose", false, "also list routes that match, with timings")
	flag.Parse()

	client := apicompare.NewClient(*adminKey)
	flaskURL, goURL := strings.TrimRight(*flaskBase, "/"), strings.TrimRight(*goBase, "/")
	routes := apicompare.Discover(client, flaskURL, *perLevel, *perList)
	fmt.Fprintf(os.Stderr, "Comparing %d route(s)\n", len(routes))
	results := make([]apicompare.Result, 0, len(routes))
	for index, route := range routes {
		results = append(results, apicompare.Compare(client, flaskURL, goURL, route))
		if (index+1)%25 == 0 {
			fmt.Fprintf(os.Stderr, "  %d / %d\n", index+1, len(routes))
		}
	}
	report, allSame := apicompare.Report(results, *verbose)
	fmt.Print(report)
	if !allSame {
		os.Exit(1)
	}
}
