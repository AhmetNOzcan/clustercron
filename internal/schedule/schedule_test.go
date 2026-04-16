package schedule

import (
	"testing"
	"time"
)

func TestNextFireTime_EveryFiveMinutes(t *testing.T) {
	ref := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	next, err := NextFireTime("*/5 * * * *", ref)
	if err != nil {
		t.Fatalf("unexptected error: %v", err)
	}

	want := time.Date(2026, 4, 16, 10, 5, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("got %v, want %v", next, want)
	}
}

func TestNextFireTime_DailyAtNine(t *testing.T) {
	// 10:30 AM — next 9 AM is tomorrow
	ref := time.Date(2026, 4, 16, 10, 30, 0, 0, time.UTC)

	next, err := NextFireTime("0 9 * * *", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

func TestNextFireTime_Descriptor(t *testing.T) {
	ref := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)

	next, err := NextFireTime("@hourly", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := time.Date(2026, 4, 16, 11, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

func TestNextFireTime_WeekdaysOnly(t *testing.T) {
	// April 18, 2026 is a Saturday
	ref := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)

	next, err := NextFireTime("0 9 * * 1-5", ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Next Monday is April 20
	want := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("got %v, want %v", next, want)
	}
}

func TestNextFireTime_InvalidExpr(t *testing.T) {
	_, err := NextFireTime("not a cron", time.Now())
	if err == nil {
		t.Fatal("expected error for invalid expression, got nil")
	}
}

func TestValidate_Valid(t *testing.T) {
	cases := []string{
		"*/5 * * * *",
		"0 9 * * 1-5",
		"@hourly",
		"@every 30s",
		"0 0 1 * *",
	}
	for _, expr := range cases {
		if err := Validate(expr); err != nil {
			t.Errorf("Validate(%q) returned error: %v", expr, err)
		}
	}
}

func TestValidate_Invalid(t *testing.T) {
	cases := []string{
		"",
		"not valid",
		"* * *",
		"60 * * * *",
	}
	for _, expr := range cases {
		if err := Validate(expr); err == nil {
			t.Errorf("Validate(%q) expected error, got nil", expr)
		}
	}
}
