package scrape

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/tool/web"
)

func TestScrape(t *testing.T) {
	tests := []struct {
		url string
	}{
		{"https://fortwoplz.com/planning-a-trip-to-california/"},
		// {"https://www.myglobalviewpoint.com/california-road-trip-itinerary/"},
		// {"https://wanderlog.com/tp/90405/california-trip-planner"},
		// {"https://www.visitcalifornia.com/trip-planning/travel-tips/"},
		// {"https://alicesadventuresonearth.com/7-day-california-national-parks-road-trip-route/"},
	}

	cli, err := New()
	if err != nil {
		t.Fatalf("failed to create scraper: %v", err)
	}
	cli.MaxDepth = 1
	cli.MaxLink = 1
	cli.Parallels = 2
	cli.Delay = 1
	cli.Async = true

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			cli.UserAgent = web.UserAgent()
			result, err := cli.Scrape(ctx, test.url)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			t.Logf("result: %s\n", result)
		})
	}
}

func TestFetch(t *testing.T) {
	tests := []struct {
		url string
	}{
		{"https://fortwoplz.com/planning-a-trip-to-california/"},
		// {"https://www.myglobalviewpoint.com/california-road-trip-itinerary/"},
		// {"https://wanderlog.com/tp/90405/california-trip-planner"},
		// {"https://www.visitcalifornia.com/trip-planning/travel-tips/"},
		// {"https://alicesadventuresonearth.com/7-day-california-national-parks-road-trip-route/"},
	}

	cli, err := New()
	if err != nil {
		t.Fatalf("failed to create scraper: %v", err)
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.url, func(t *testing.T) {
			cli.UserAgent = web.UserAgent()
			result, err := cli.Fetch(ctx, test.url)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			t.Logf("result: %s\n", result)
		})
	}
}
