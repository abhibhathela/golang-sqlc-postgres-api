package helpers

import (
	"fmt"
	"time"

	"github.com/robfig/cron"
)

func IsValidDateToUnlockReward(schedule string) (bool, error) {
	parsedSchedule, err := cron.ParseStandard(schedule)
	if err != nil {
		return false, fmt.Errorf("error while parsing the schedule")
	}

	nextTime := parsedSchedule.Next(time.Now())

	year := nextTime.Year()
	month := nextTime.Month()
	day := nextTime.Day()

	cYear := time.Now().Year()
	cMonth := time.Now().Month()
	cDay := time.Now().Day()

	// if date month year match then print "today"
	if year == cYear && month == cMonth && day == cDay {
		return true, nil
	}

	fmt.Println("nextTime", nextTime, year == cYear && month == cMonth && day == cDay)

	return false, nil
}
