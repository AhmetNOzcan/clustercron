package schedule

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func NextFireTime(expr string, after time.Time) (time.Time, error) {
	sched, err := parser.Parse(expr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return sched.Next(after), nil
}

func Validate(expr string) error {
	_, err := parser.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", expr, err)
	}
	return nil
}
