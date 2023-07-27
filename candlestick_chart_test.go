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
		time.Date(2023, time.July, 27, 0, 0, 0, 0, time.UTC): {
			Start: time.Date(2023, time.July, 27, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2023, time.July, 27, 24, 0, 0, 0, time.UTC),
		},
	}
	chart, err := NewCandlestickChart(PeriodMinute, m, time.UTC)
	if err != nil {
		log.Fatalln(err)
	}
	defer chart.Stop()
	chart.WatchFunc(func(c *Candlestick) {
		b, _ := json.Marshal(c)
		log.Printf("watch:%s\n", string(b))
	})
	txs := []map[string]any{
		{
			"delay": 10, "value": 12, "volume": 1,
		},
		{
			"delay": 22, "value": 23, "volume": 1,
		},
		{
			"delay": 31, "value": 45, "volume": 2,
		},
		{
			"delay": 44, "value": 15, "volume": 6,
		},
		{
			"delay": 55, "value": 33, "volume": 34,
		},
		{
			"delay": 66, "value": 47, "volume": 347,
		},
		{
			"delay": 71, "value": 52, "volume": 42,
		},
		{
			"delay": 92, "value": 12, "volume": 14,
		},
		{
			"delay": 102, "value": 24, "volume": 246,
		},
		{
			"delay": 113, "value": 23, "volume": 35,
		},
	}
	for _, tx := range txs {
		go trade(chart, tx)
	}
	time.Sleep(5 * time.Minute)
	b, _ := json.Marshal(chart.candlesticks)
	fmt.Println(string(b))
}

func trade(chart *CandlestickChart, tx map[string]any) {
	delay := tx["delay"].(int)
	value := tx["value"].(int)
	volume := tx["volume"].(int)
	time.Sleep(time.Duration(delay) * time.Second)
	_ = chart.AddTrade(time.Now().UTC(), decimal.NewFromInt(int64(value)), int64(volume))
}

func TestToX(t *testing.T) {
	ti := time.Now()
	log.Println(ti.Truncate(time.Minute))
	log.Println(ti.Truncate(5 * time.Minute))
	log.Println(ti.Truncate(10 * time.Minute))
	log.Println(ti.Truncate(15 * time.Minute))
	log.Println(ti.Truncate(30 * time.Minute))
	log.Println(ti.Truncate(time.Hour))
	log.Println(time.Date(ti.Year(), ti.Month(), ti.Day(), 0, 0, 0, 0, ti.Location()))
	offset := ti.Weekday() - 1
	if offset < 0 {
		offset = 6
	}
	nt := time.Date(ti.Year(), ti.Month(), ti.Day(), 0, 0, 0, 0, ti.Location())
	nt = nt.Add(time.Duration(-offset) * 24 * time.Hour)
	log.Println(nt)
	log.Println(time.Date(ti.Year(), ti.Month(), 1, 0, 0, 0, 0, ti.Location()))
	log.Println(time.Date(ti.Year(), 1, 1, 0, 0, 0, 0, ti.Location()))
}
