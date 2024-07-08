package chrome

import (
	"fmt"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/mahendrapaipuri/grafana-dashboard-reporter-app/pkg/plugin/internal/config"
	"golang.org/x/net/context"
)

/*
	This file contains chromedp package related helper functions.
	Sources:
		- https://github.com/chromedp/chromedp/issues/1044
		- https://github.com/chromedp/chromedp/issues/431#issuecomment-592950397
		- https://github.com/chromedp/chromedp/issues/87
		- https://github.com/chromedp/examples/tree/master
*/

// PDFOptions contains the templated HTML Body, Header and Footer strings
type PDFOptions struct {
	Header string
	Body   string
	Footer string

	Orientation string
}

type Instance interface {
	NewTab(ctx context.Context, logger log.Logger, conf *config.Config) *Tab
	Close()
}

// enableLifeCycleEvents enables the chromedp life cycle events
func enableLifeCycleEvents() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		err := page.Enable().Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable page: %w", err)
		}

		err = page.SetLifecycleEventsEnabled(true).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to enable lifecycle events: %w", err)
		}

		return nil
	}
}

// navigateAndWaitFor navigates the browser to the given URL and waits for it to
// load until a given event occurs
func navigateAndWaitFor(url string, eventName string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		_, _, _, err := page.Navigate(url).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to navigate: %w", err)
		}
		return waitFor(ctx, eventName)
	}
}

// waitFor blocks until eventName is received.
// Examples of events you can wait for:
//
//	init, DOMContentLoaded, firstPaint,
//	firstContentfulPaint, firstImagePaint,
//	firstMeaningfulPaintCandidate,
//	load, networkAlmostIdle, firstMeaningfulPaint, networkIdle
//
// This is not super reliable, I've already found incidental cases where
// networkIdle was sent before load. It's probably smart to see how
// puppeteer implements this exactly.
func waitFor(ctx context.Context, eventName string) error {
	ch := make(chan struct{})
	cctx, cancel := context.WithCancel(ctx)
	chromedp.ListenTarget(cctx, func(ev interface{}) {
		switch e := ev.(type) {
		case *page.EventLifecycleEvent:
			if e.Name == eventName {
				cancel()
				close(ch)
			}
		}
	})
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetHeaders returns a task list that sets the passed headers.
func SetHeaders(u string, headers map[string]any) chromedp.Tasks {
	return chromedp.Tasks{
		network.Enable(),
		network.SetExtraHTTPHeaders(headers),
		enableLifeCycleEvents(),
		navigateAndWaitFor(u, "networkIdle"),
	}
}
