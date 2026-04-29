package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func effectiveMetricsTimeRangeString(c *gin.Context) (start, end string, err error) {
	ds, de := c.Query("startDate"), c.Query("endDate")
	if ds == "" && de == "" {
		now := time.Now().UTC()
		return now.AddDate(0, 0, -30).Format(dbTimestampFormat), now.Format(dbTimestampFormat), nil
	}
	s, err := parseDateParam(ds)
	if err != nil {
		return "", "", err
	}
	e, err := parseDateParam(de)
	if err != nil {
		return "", "", err
	}
	if s == "" {
		now := time.Now().UTC()
		s = now.AddDate(0, 0, -30).Format(dbTimestampFormat)
	}
	if e == "" {
		e = time.Now().UTC().Format(dbTimestampFormat)
	}
	return s, e, nil
}

func loadDashboardSettings() (preferredRange, lengthUnit, tempUnit string, err error) {
	err = db.QueryRow(`SELECT preferred_range, unit_of_length, unit_of_temperature FROM settings LIMIT 1`).Scan(&preferredRange, &lengthUnit, &tempUnit)
	if err != nil {
		return
	}
	if preferredRange == "" {
		preferredRange = "rated"
	}
	return
}

func preferredRangeFromQuery(c *gin.Context, defaultPR string) string {
	q := strings.ToLower(strings.TrimSpace(c.Query("preferredRange")))
	switch q {
	case "ideal", "rated":
		return q
	default:
		return defaultPR
	}
}

func validatePreferredRangeColumn(pref string) (string, error) {
	switch pref {
	case "ideal", "rated":
		return pref, nil
	default:
		return "", fmt.Errorf("preferredRange must be ideal or rated")
	}
}

func validateStatisticsPeriod(p string) (string, error) {
	switch p {
	case "day", "week", "month", "year":
		return p, nil
	default:
		return "", fmt.Errorf("period must be day, week, month, or year")
	}
}

func safeTimezoneForSQL(c *gin.Context) string {
	tz := strings.TrimSpace(c.Query("timezone"))
	if tz != "" {
		if _, err := time.LoadLocation(tz); err == nil {
			return tz
		}
	}
	if appUsersTimezone != nil {
		return appUsersTimezone.String()
	}
	return "UTC"
}

func validateProjectedInterval(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "1 hour", nil
	}
	allowed := map[string]bool{
		"5 minutes": true, "10 minutes": true, "15 minutes": true, "30 minutes": true,
		"1 hour": true, "3 hours": true, "6 hours": true, "12 hours": true, "1 day": true,
	}
	if allowed[s] {
		return s, nil
	}
	return "", fmt.Errorf("invalid interval: use e.g. 5 minutes, 1 hour, 1 day")
}
