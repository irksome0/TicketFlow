package sla

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ticketflow-api/internal/models"
)

type Status string

const (
	StatusWithin   Status = "Within SLA"
	StatusPaused   Status = "Paused"
	StatusMet      Status = "Met"
	StatusBreached Status = "Breached"
)

type Config struct {
	LocationName  string
	BusinessStart time.Duration
	BusinessEnd   time.Duration
	Holidays      []string
	PolicyLimits  map[models.TicketPriority]time.Duration
}

type Engine struct {
	location      *time.Location
	businessStart time.Duration
	businessEnd   time.Duration
	holidays      map[string]struct{}
	policyLimits  map[models.TicketPriority]time.Duration
}

type Result struct {
	Limit             time.Duration
	ActiveDuration    time.Duration
	PausedDuration    time.Duration
	RemainingDuration time.Duration
	Status            Status
	DueAt             *time.Time
	BreachedAt        *time.Time
	CalendarTimezone  string
	BusinessHours     string
}

func DefaultConfig() Config {
	return Config{
		LocationName:  "Europe/Kyiv",
		BusinessStart: 9 * time.Hour,
		BusinessEnd:   18 * time.Hour,
		PolicyLimits: map[models.TicketPriority]time.Duration{
			models.PriorityHigh:   8 * time.Hour,
			models.PriorityMedium: 24 * time.Hour,
			models.PriorityLow:    72 * time.Hour,
		},
	}
}

func NewEngine(cfg Config) (*Engine, error) {
	defaults := DefaultConfig()
	if strings.TrimSpace(cfg.LocationName) == "" {
		cfg.LocationName = defaults.LocationName
	}
	if cfg.BusinessStart == 0 {
		cfg.BusinessStart = defaults.BusinessStart
	}
	if cfg.BusinessEnd == 0 {
		cfg.BusinessEnd = defaults.BusinessEnd
	}
	if cfg.PolicyLimits == nil {
		cfg.PolicyLimits = defaults.PolicyLimits
	}

	if cfg.BusinessEnd <= cfg.BusinessStart {
		return nil, fmt.Errorf("business end must be later than business start")
	}

	location, err := time.LoadLocation(cfg.LocationName)
	if err != nil {
		return nil, fmt.Errorf("load SLA timezone %q: %w", cfg.LocationName, err)
	}

	holidays := make(map[string]struct{}, len(cfg.Holidays))
	for _, holiday := range cfg.Holidays {
		value := strings.TrimSpace(holiday)
		if value == "" {
			continue
		}
		parsed, err := time.ParseInLocation("2006-01-02", value, location)
		if err != nil {
			return nil, fmt.Errorf("parse SLA holiday %q: %w", value, err)
		}
		holidays[dateKey(parsed, location)] = struct{}{}
	}

	policies := make(map[models.TicketPriority]time.Duration, len(defaults.PolicyLimits))
	for priority, limit := range defaults.PolicyLimits {
		policies[priority] = limit
	}
	for priority, limit := range cfg.PolicyLimits {
		if limit > 0 {
			policies[priority] = limit
		}
	}

	return &Engine{
		location:      location,
		businessStart: cfg.BusinessStart,
		businessEnd:   cfg.BusinessEnd,
		holidays:      holidays,
		policyLimits:  policies,
	}, nil
}

func MustDefaultEngine() *Engine {
	engine, err := NewEngine(DefaultConfig())
	if err != nil {
		panic(err)
	}
	return engine
}

func (e *Engine) Calculate(ticket models.Ticket, now time.Time) Result {
	if e == nil {
		e = MustDefaultEngine()
	}

	limit := e.limit(ticket.Priority)
	history := normalizedHistory(ticket)

	var active time.Duration
	var paused time.Duration
	var breachedAt *time.Time

	for i, item := range history {
		start := item.CreatedAt
		if start.Before(ticket.CreatedAt) {
			start = ticket.CreatedAt
		}

		end := now
		if i+1 < len(history) {
			end = history[i+1].CreatedAt
		}
		if !end.After(start) {
			continue
		}

		switch {
		case isActiveStatus(item.ToStatus):
			segment := e.BusinessDuration(start, end)
			if breachedAt == nil && active < limit && active+segment > limit {
				value := e.AddBusinessDuration(start, limit-active)
				breachedAt = &value
			}
			active += segment
		case isPausedStatus(item.ToStatus):
			paused += e.BusinessDuration(start, end)
		}
	}

	remaining := limit - active
	if remaining < 0 {
		remaining = 0
	}

	status := e.status(ticket.Status, active, limit)
	var dueAt *time.Time
	if breachedAt != nil {
		dueAt = breachedAt
	} else if isActiveStatus(ticket.Status) && remaining > 0 {
		value := e.AddBusinessDuration(now, remaining)
		dueAt = &value
	}

	return Result{
		Limit:             limit,
		ActiveDuration:    active,
		PausedDuration:    paused,
		RemainingDuration: remaining,
		Status:            status,
		DueAt:             dueAt,
		BreachedAt:        breachedAt,
		CalendarTimezone:  e.location.String(),
		BusinessHours:     e.BusinessHoursLabel(),
	}
}

func (e *Engine) BusinessDuration(start, end time.Time) time.Duration {
	if !end.After(start) {
		return 0
	}

	localStart := start.In(e.location)
	localEnd := end.In(e.location)
	total := time.Duration(0)

	for day := startOfDay(localStart, e.location); !day.After(localEnd); day = day.AddDate(0, 0, 1) {
		if !e.isBusinessDay(day) {
			continue
		}

		windowStart, windowEnd := e.businessWindow(day)
		intervalStart := maxTime(localStart, windowStart)
		intervalEnd := minTime(localEnd, windowEnd)
		if intervalEnd.After(intervalStart) {
			total += intervalEnd.Sub(intervalStart)
		}
	}

	return total
}

func (e *Engine) AddBusinessDuration(start time.Time, duration time.Duration) time.Time {
	current := start.In(e.location)
	remaining := duration
	if remaining <= 0 {
		return current.UTC()
	}

	for {
		day := startOfDay(current, e.location)
		if !e.isBusinessDay(day) {
			current = day.AddDate(0, 0, 1)
			continue
		}

		windowStart, windowEnd := e.businessWindow(day)
		if current.Before(windowStart) {
			current = windowStart
		}
		if !current.Before(windowEnd) {
			current = day.AddDate(0, 0, 1)
			continue
		}

		available := windowEnd.Sub(current)
		if remaining <= available {
			return current.Add(remaining).UTC()
		}

		remaining -= available
		current = day.AddDate(0, 0, 1)
	}
}

func (e *Engine) BusinessHoursLabel() string {
	return fmt.Sprintf(
		"Mon-Fri %s-%s",
		formatClock(e.businessStart),
		formatClock(e.businessEnd),
	)
}

func (e *Engine) limit(priority models.TicketPriority) time.Duration {
	if limit, ok := e.policyLimits[priority]; ok {
		return limit
	}
	return e.policyLimits[models.PriorityMedium]
}

func (e *Engine) status(status models.TicketStatus, active, limit time.Duration) Status {
	if active > limit {
		return StatusBreached
	}
	if isPausedStatus(status) {
		return StatusPaused
	}
	if isTerminalStatus(status) {
		return StatusMet
	}
	return StatusWithin
}

func (e *Engine) isBusinessDay(day time.Time) bool {
	weekday := day.Weekday()
	if weekday == time.Saturday || weekday == time.Sunday {
		return false
	}
	_, holiday := e.holidays[dateKey(day, e.location)]
	return !holiday
}

func (e *Engine) businessWindow(day time.Time) (time.Time, time.Time) {
	start := startOfDay(day, e.location).Add(e.businessStart)
	end := startOfDay(day, e.location).Add(e.businessEnd)
	return start, end
}

func normalizedHistory(ticket models.Ticket) []models.TicketStatusHistory {
	history := make([]models.TicketStatusHistory, len(ticket.StatusHistory))
	copy(history, ticket.StatusHistory)
	sort.Slice(history, func(i, j int) bool {
		return history[i].CreatedAt.Before(history[j].CreatedAt)
	})

	if len(history) == 0 {
		return []models.TicketStatusHistory{
			{
				TicketID:  ticket.ID,
				ToStatus:  ticket.Status,
				CreatedAt: ticket.CreatedAt,
			},
		}
	}

	return history
}

func isActiveStatus(status models.TicketStatus) bool {
	switch status {
	case models.StatusNew, models.StatusInProgress, models.StatusReopened:
		return true
	default:
		return false
	}
}

func isPausedStatus(status models.TicketStatus) bool {
	switch status {
	case models.StatusPending, models.StatusWaiting, models.StatusOnHold:
		return true
	default:
		return false
	}
}

func isTerminalStatus(status models.TicketStatus) bool {
	switch status {
	case models.StatusResolved, models.StatusClosed:
		return true
	default:
		return false
	}
}

func startOfDay(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
}

func dateKey(value time.Time, location *time.Location) string {
	return value.In(location).Format("2006-01-02")
}

func formatClock(value time.Duration) string {
	totalMinutes := int(value.Minutes())
	return fmt.Sprintf("%02d:%02d", totalMinutes/60, totalMinutes%60)
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
