package automation

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gotify/server/v3/model"
	"github.com/gotify/server/v3/security"
)

type feedDocument struct {
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title string `xml:"title"`
			Link string `xml:"link"`
			GUID string `xml:"guid"`
			Description string `xml:"description"`
			PubDate string `xml:"pubDate"`
		} `xml:"item"`
	} `xml:"channel"`
	Entries []struct {
		Title string `xml:"title"`
		ID string `xml:"id"`
		Summary string `xml:"summary"`
		Content string `xml:"content"`
		Updated string `xml:"updated"`
		Links []struct { Href string `xml:"href,attr"` } `xml:"link"`
	} `xml:"entry"`
}

func (e *Engine) runRSSLoop(ctx context.Context, integration *model.RSSIntegration) {
	key := fmt.Sprintf("integration:rss:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 3*time.Minute, func(leased context.Context) error {
			e.setIntegrationStatus("rss", integration.ID, "connected", "", true, false)
			interval := time.Duration(integration.PollMinutes) * time.Minute
			if interval < time.Minute { interval = 5 * time.Minute }
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			if err := e.pollRSS(leased, integration); err != nil { return err }
			for {
				select {
				case <-leased.Done(): return leased.Err()
				case <-ticker.C:
					if err := e.pollRSS(leased, integration); err != nil { return err }
				}
			}
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("rss", integration.ID, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("rss", integration.ID, "reconnecting", err.Error(), false, false)
		}
		select {
		case <-ctx.Done(): return
		case <-time.After(15*time.Second):
		}
	}
}

func (e *Engine) pollRSS(ctx context.Context, integration *model.RSSIntegration) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, integration.URL, nil)
	if err != nil { return err }
	req.Header.Set("User-Agent", "Gotify-MU RSS Monitor")
	if integration.ETag != "" { req.Header.Set("If-None-Match", integration.ETag) }
	if integration.LastModified != "" { req.Header.Set("If-Modified-Since", integration.LastModified) }

	resp, err := (&http.Client{Timeout:20*time.Second}).Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		e.setIntegrationStatus("rss", integration.ID, "connected", "", false, true)
		return nil
	}
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("feed returned HTTP %d", resp.StatusCode) }

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil { return err }
	var doc feedDocument
	if err := xml.Unmarshal(body, &doc); err != nil { return fmt.Errorf("invalid RSS/Atom feed: %w", err) }

	type entry struct{ key, title, body string }
	entries := make([]entry, 0)
	for _, item := range doc.Channel.Items {
		key := strings.TrimSpace(item.GUID)
		if key == "" { key = strings.TrimSpace(item.Link) }
		if key == "" { key = item.Title + "|" + item.PubDate }
		message := strings.TrimSpace(item.Description)
		if item.Link != "" { message = strings.TrimSpace(message + "\n\n" + item.Link) }
		entries = append(entries, entry{key:key, title:item.Title, body:message})
	}
	for _, item := range doc.Entries {
		link := ""
		if len(item.Links) > 0 { link = item.Links[0].Href }
		key := strings.TrimSpace(item.ID)
		if key == "" { key = link }
		if key == "" { key = item.Title + "|" + item.Updated }
		message := strings.TrimSpace(item.Summary)
		if message == "" { message = strings.TrimSpace(item.Content) }
		if link != "" { message = strings.TrimSpace(message + "\n\n" + link) }
		entries = append(entries, entry{key:key, title:item.Title, body:message})
	}

	for i := len(entries)-1; i >= 0; i-- {
		item := entries[i]
		if item.key == "" { continue }
		trigger := fmt.Sprintf("rss:%d:%s", integration.ID, security.StableHash(item.key))
		if existing, loadErr := e.db.GetAutomationRunByTrigger(trigger); loadErr != nil {
			return loadErr
		} else if existing != nil {
			continue
		}
		run := &model.AutomationRun{Kind:"rss", ObjectID:integration.ID, TriggerKey:trigger, Status:"running", StartedAt:time.Now()}
		created, createErr := e.db.CreateAutomationRun(run)
		if createErr != nil { return createErr }
		if !created { continue }

		title := strings.TrimSpace(integration.TitlePrefix + " " + item.title)
		msg, pubErr := e.publishWithKey(integration.ApplicationID, title, item.body, 0, trigger, 0)
		done := time.Now()
		run.FinishedAt = &done
		if pubErr != nil {
			run.Status = "failed"
			run.Error = pubErr.Error()
		} else {
			run.Status = "completed"
			run.MessageID = msg.ID
		}
		_ = e.db.SaveAutomationRun(run)
	}
	integration.ETag = resp.Header.Get("ETag")
	integration.LastModified = resp.Header.Get("Last-Modified")
	if err := e.db.SaveRSSIntegration(integration); err != nil { return err }
	e.setIntegrationStatus("rss", integration.ID, "connected", "", false, true)
	return nil
}

type calendarEvent struct {
	UID string
	Summary string
	Description string
	Location string
	Start time.Time
}

func unfoldICal(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	input := strings.Split(raw, "\n")
	out := make([]string, 0, len(input))
	for _, line := range input {
		if (strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")) && len(out) > 0 {
			out[len(out)-1] += strings.TrimLeft(line, " \t")
		} else {
			out = append(out, line)
		}
	}
	return out
}

func parseICalTime(key, value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	loc := time.UTC
	if idx := strings.Index(key, "TZID="); idx >= 0 {
		tz := key[idx+5:]
		if semi := strings.IndexByte(tz, ';'); semi >= 0 { tz = tz[:semi] }
		if parsed, err := time.LoadLocation(tz); err == nil { loc = parsed }
	}
	if parsed, err := time.Parse("20060102T150405Z", value); err == nil { return parsed, nil }
	if parsed, err := time.ParseInLocation("20060102T150405", value, loc); err == nil { return parsed, nil }
	if parsed, err := time.ParseInLocation("20060102", value, loc); err == nil { return parsed, nil }
	return time.Time{}, errors.New("unsupported DTSTART")
}

func parseICal(raw string) []calendarEvent {
	var events []calendarEvent
	var current *calendarEvent
	for _, line := range unfoldICal(raw) {
		switch strings.TrimSpace(line) {
		case "BEGIN:VEVENT":
			current = &calendarEvent{}
		case "END:VEVENT":
			if current != nil && !current.Start.IsZero() { events = append(events, *current) }
			current = nil
		default:
			if current == nil { continue }
			idx := strings.IndexByte(line, ':')
			if idx < 0 { continue }
			key, value := line[:idx], line[idx+1:]
			value = strings.ReplaceAll(strings.ReplaceAll(value, "\\n", "\n"), "\\;", ";")
			base := key
			if semi := strings.IndexByte(base, ';'); semi >= 0 { base = base[:semi] }
			switch base {
			case "UID": current.UID = value
			case "SUMMARY": current.Summary = value
			case "DESCRIPTION": current.Description = value
			case "LOCATION": current.Location = value
			case "DTSTART":
				if parsed, err := parseICalTime(key, value); err == nil { current.Start = parsed }
			}
		}
	}
	return events
}

func (e *Engine) runCalendarLoop(ctx context.Context, integration *model.CalendarIntegration) {
	key := fmt.Sprintf("integration:calendar:%d", integration.ID)
	for {
		if ctx.Err() != nil { return }
		err := e.runWithLease(ctx, key, 3*time.Minute, func(leased context.Context) error {
			e.setIntegrationStatus("calendar", integration.ID, "connected", "", true, false)
			interval := time.Duration(integration.PollMinutes) * time.Minute
			if interval < time.Minute { interval = 5 * time.Minute }
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			if err := e.pollCalendar(leased, integration); err != nil { return err }
			for {
				select {
				case <-leased.Done(): return leased.Err()
				case <-ticker.C:
					if err := e.pollCalendar(leased, integration); err != nil { return err }
				}
			}
		})
		if ctx.Err() != nil { return }
		if errors.Is(err, errLeaseUnavailable) {
			e.setIntegrationStatus("calendar", integration.ID, "standby", "", false, false)
		} else if err != nil {
			e.setIntegrationStatus("calendar", integration.ID, "reconnecting", err.Error(), false, false)
		}
		select {
		case <-ctx.Done(): return
		case <-time.After(15*time.Second):
		}
	}
}

func (e *Engine) pollCalendar(ctx context.Context, integration *model.CalendarIntegration) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, integration.URL, nil)
	if err != nil { return err }
	resp, err := (&http.Client{Timeout:20*time.Second}).Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return fmt.Errorf("calendar returned HTTP %d", resp.StatusCode) }
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil { return err }

	events := parseICal(string(body))
	now := time.Now()
	pollMinutes := integration.PollMinutes
	if pollMinutes < 5 { pollMinutes = 5 }
	windowEnd := now.Add(time.Duration(pollMinutes) * time.Minute)

	for _, event := range events {
		triggerAt := event.Start.Add(-time.Duration(integration.AdvanceMinutes) * time.Minute)
		if triggerAt.After(windowEnd) || triggerAt.After(now) || event.Start.Before(now) { continue }
		key := fmt.Sprintf("calendar:%d:%s:%d", integration.ID, security.StableHash(event.UID), event.Start.Unix())
		if existing, loadErr := e.db.GetAutomationRunByTrigger(key); loadErr != nil {
			return loadErr
		} else if existing != nil {
			continue
		}
		run := &model.AutomationRun{Kind:"calendar", ObjectID:integration.ID, TriggerKey:key, Status:"running", StartedAt:now}
		created, createErr := e.db.CreateAutomationRun(run)
		if createErr != nil { return createErr }
		if !created { continue }

		message := event.Description
		if event.Location != "" { message = strings.TrimSpace(message + "\n\nLocation: " + event.Location) }
		message = strings.TrimSpace(message + "\n\nStarts: " + event.Start.Local().Format(time.RFC1123))
		msg, pubErr := e.publishWithKey(integration.ApplicationID, event.Summary, message, 0, key, 0)
		done := time.Now()
		run.FinishedAt = &done
		if pubErr != nil {
			run.Status = "failed"
			run.Error = pubErr.Error()
		} else {
			run.Status = "completed"
			run.MessageID = msg.ID
		}
		_ = e.db.SaveAutomationRun(run)
	}
	e.setIntegrationStatus("calendar", integration.ID, "connected", "", false, true)
	return nil
}
