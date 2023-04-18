package main

import (
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
)

func main() {
	times, err := cron.ParseStandard("0 0 * * *")
	if err != nil {
		panic(err)
	}

	nextTime := times.Next(time.Now())
	fmt.Println(nextTime)

	fmt.Println("time.Now():", time.Now())
	fmt.Println(nextTime.Compare(time.Now()))

	//! if date month year match then print "today"
	if nextTime.Year() == time.Now().Year() && nextTime.Month() == time.Now().Month() && nextTime.Day() == time.Now().Day() {
		fmt.Println("today")
	}

}
