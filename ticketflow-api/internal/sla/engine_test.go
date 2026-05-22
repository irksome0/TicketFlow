package sla

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"ticketflow-api/internal/models"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()

	engine, err := NewEngine(DefaultConfig())
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return engine
}

func kyivTime(t *testing.T, year int, month time.Month, day, hour, minute int) time.Time {
	t.Helper()

	location, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	return time.Date(year, month, day, hour, minute, 0, 0, location)
}

func TestBusinessDurationSkipsWeekend(t *testing.T) {
	engine := testEngine(t)

	start := kyivTime(t, 2026, time.May, 22, 16, 0)
	end := kyivTime(t, 2026, time.May, 25, 10, 0)

	got := engine.BusinessDuration(start, end)
	want := 3 * time.Hour
	if got != want {
		t.Fatalf("BusinessDuration = %v, want %v", got, want)
	}
}

func TestCalculateExcludesPausedIntervals(t *testing.T) {
	engine := testEngine(t)
	ticketID := uuid.New()
	userID := uuid.New()

	ticket := models.Ticket{
		ID:        ticketID,
		Status:    models.StatusInProgress,
		Priority:  models.PriorityMedium,
		CreatedAt: kyivTime(t, 2026, time.May, 25, 10, 0),
		StatusHistory: []models.TicketStatusHistory{
			{
				TicketID:  ticketID,
				ChangedBy: userID,
				ToStatus:  models.StatusNew,
				CreatedAt: kyivTime(t, 2026, time.May, 25, 10, 0),
			},
			{
				TicketID:  ticketID,
				ChangedBy: userID,
				ToStatus:  models.StatusWaiting,
				CreatedAt: kyivTime(t, 2026, time.May, 25, 12, 0),
			},
			{
				TicketID:  ticketID,
				ChangedBy: userID,
				ToStatus:  models.StatusInProgress,
				CreatedAt: kyivTime(t, 2026, time.May, 26, 10, 0),
			},
		},
	}

	result := engine.Calculate(ticket, kyivTime(t, 2026, time.May, 26, 12, 0))

	if result.ActiveDuration != 4*time.Hour {
		t.Fatalf("ActiveDuration = %v, want 4h", result.ActiveDuration)
	}
	if result.PausedDuration != 7*time.Hour {
		t.Fatalf("PausedDuration = %v, want 7h", result.PausedDuration)
	}
	if result.Status != StatusWithin {
		t.Fatalf("Status = %s, want %s", result.Status, StatusWithin)
	}
}

func TestCalculateDetectsBusinessHourBreach(t *testing.T) {
	engine := testEngine(t)
	ticketID := uuid.New()
	userID := uuid.New()

	ticket := models.Ticket{
		ID:        ticketID,
		Status:    models.StatusInProgress,
		Priority:  models.PriorityHigh,
		CreatedAt: kyivTime(t, 2026, time.May, 25, 9, 0),
		StatusHistory: []models.TicketStatusHistory{
			{
				TicketID:  ticketID,
				ChangedBy: userID,
				ToStatus:  models.StatusNew,
				CreatedAt: kyivTime(t, 2026, time.May, 25, 9, 0),
			},
			{
				TicketID:  ticketID,
				ChangedBy: userID,
				ToStatus:  models.StatusInProgress,
				CreatedAt: kyivTime(t, 2026, time.May, 25, 10, 0),
			},
		},
	}

	result := engine.Calculate(ticket, kyivTime(t, 2026, time.May, 26, 10, 0))

	if result.Status != StatusBreached {
		t.Fatalf("Status = %s, want %s", result.Status, StatusBreached)
	}
	if result.BreachedAt == nil {
		t.Fatal("BreachedAt is nil")
	}

	wantBreach := kyivTime(t, 2026, time.May, 25, 17, 0).UTC()
	if !result.BreachedAt.Equal(wantBreach) {
		t.Fatalf("BreachedAt = %s, want %s", result.BreachedAt, wantBreach)
	}
}
