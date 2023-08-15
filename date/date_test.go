package date

import (
	"testing"

	"github.com/DoOR-Team/goutils/log"
)

func TestGetFirstDateOfWeek(t *testing.T) {
	log.Info("本周第一天日期", GetFirstDateOfWeek())
	log.Info("本周最后一天日期", GetLastDateOfWeek())
	log.Info("上周第一天日期", GetLastWeekFirstDate())
	log.Info("上周最后一天日期", GetLastWeekLastDate())
}
