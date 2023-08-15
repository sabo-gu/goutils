package date

import (
	"time"
)

func Today() time.Time {
	today, _ := time.ParseInLocation("2006-01-02", time.Now().Local().Format("2006-01-02"), time.Local)
	return today
}

func Yestoday() time.Time {
	today := Today().Unix() - 1
	yestorday, _ := time.ParseInLocation("2006-01-02", time.Unix(today, 0).Format("2006-01-02"), time.Local)
	return yestorday
}

func DayStart(t time.Time) time.Time {
	yestorday, _ := time.ParseInLocation("2006-01-02", t.Format("2006-01-02"), time.Local)
	return yestorday
}

/**
获取本周周一的日期
*/
func GetFirstDateOfWeek() (weekMonday string) {
	now := time.Now()

	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekMonday = weekStartDate.Format("2006-01-02")
	return
}

/**
获取本周周日的日期
*/
func GetLastDateOfWeek() (weekMonday string) {
	now := time.Now()

	offset := int(time.Sunday - now.Weekday())
	if offset < 0 {
		offset += 7
	}

	weekStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekMonday = weekStartDate.Format("2006-01-02")
	return
}

/**
获取上周的周一日期
*/
func GetLastWeekFirstDate() (weekMonday string) {
	thisWeekMonday := GetFirstDateOfWeek()
	TimeMonday, _ := time.Parse("2006-01-02", thisWeekMonday)
	lastWeekMonday := TimeMonday.AddDate(0, 0, -7)
	weekMonday = lastWeekMonday.Format("2006-01-02")
	return
}

/**
获取上周的周日日期
*/
func GetLastWeekLastDate() (weekMonday string) {
	thisWeekMonday := GetLastDateOfWeek()
	TimeMonday, _ := time.Parse("2006-01-02", thisWeekMonday)
	lastWeekMonday := TimeMonday.AddDate(0, 0, -7)
	weekMonday = lastWeekMonday.Format("2006-01-02")
	return
}
