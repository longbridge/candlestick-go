package candlestick

import (
	"encoding/json"
	"fmt"
	"log"

	// "os"
	// "os/signal"
	// "syscall"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestChart(t *testing.T) {
	m := map[time.Time]*DayTime{
		time.Date(2022, time.August, 3, 0, 0, 0, 0, time.UTC): {
			Start: time.Date(2022, time.August, 3, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2022, time.August, 3, 24, 0, 0, 0, time.UTC),
		},
	}
	chart, err := NewCandlestickChart(PeriodMinute, m)
	if err != nil {
		log.Fatalln(err)
	}
	defer chart.Stop()
	chart.WatchFunc(func(c *Candlestick) {
		b, _ := json.Marshal(c)
		fmt.Printf("watch:%s\n", string(b))
	})
	chart.AddTrade(time.Now().UTC(), decimal.NewFromInt(12), 1)

	time.Sleep(1 * time.Minute)
	chart.AddTrade(time.Now().UTC(), decimal.NewFromInt(11), 1)
	time.Sleep(2 * time.Minute)
	chart.AddTrade(time.Now().UTC(), decimal.NewFromInt(14), 1)

	b, _ := json.Marshal(chart.candlesticks)
	fmt.Println(string(b))
}
