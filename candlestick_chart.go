package candlestick

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Period uint8

const (
	PeriodMinute Period = iota
	PeriodFiveMinute
	PeriodQuarterHour
	PeriodHalfHour
	PeriodHour
	PeriodDay
	PeriodWeek
	PeriodMonth
	PeriodYear
)

// Candlestick is candlestick details
type Candlestick struct {
	Close    decimal.Decimal
	Open     decimal.Decimal
	Low      decimal.Decimal
	High     decimal.Decimal
	Volume   int64
	Turnover decimal.Decimal
	Time     time.Time
}

func NewCandle(t time.Time, value decimal.Decimal, volume int64) *Candlestick {
	return &Candlestick{
		High:   value,
		Low:    value,
		Close:  value,
		Open:   value,
		Time:   t,
		Volume: volume,
	}
}

func (candle *Candlestick) Add(value decimal.Decimal, volume int64) {
	if value.IsZero() {
		candle.Volume += volume
		return
	}
	if value.GreaterThan(candle.High) {
		candle.High = value
	}
	if value.LessThan(candle.Low) {
		candle.Low = value
	}
	candle.Volume += volume
	candle.Close = value
}

type DayTime struct {
	Start time.Time
	End   time.Time
}

type CandlestickChart struct {
	mutex            sync.Mutex
	candlesticks     []*Candlestick
	candleTimeSeries map[time.Time]*Candlestick
	timeSeries       TimeSeries
	lastCandle       *Candlestick
	currentCandle    *Candlestick
	watchFunc        func(*Candlestick)
	exit             chan struct{}
}

func NewCandlestickChart(period Period, timeRange map[time.Time]*DayTime, loc *time.Location) (*CandlestickChart, error) {
	chart := &CandlestickChart{
		candlesticks:     make([]*Candlestick, 0, 8),
		candleTimeSeries: make(map[time.Time]*Candlestick, 8),
		timeSeries: TimeSeries{
			period:    period,
			timeRange: timeRange,
			loc:       loc,
		},
		exit: make(chan struct{}, 1),
	}
	now := time.Now()
	x, err := chart.timeSeries.NextX(now)
	if err != nil {
		return nil, err
	}
	go func() {
		d := x.Sub(now)
		timer := time.NewTimer(d)
		for {
			select {
			case <-chart.exit:
				timer.Stop()
			case <-timer.C:
				now := time.Now()
				_ = chart.AddEmpty(now.Add(-time.Minute))
				nx, err := chart.timeSeries.NextX(now)
				if err != nil {
					fmt.Println(err)
					nx = now.Add(time.Hour)
				}
				nextDuration := nx.Sub(now)
				timer.Reset(nextDuration)
			}
		}
	}()

	return chart, nil
}

func (chart *CandlestickChart) AddEmpty(t time.Time) error {
	x, err := chart.timeSeries.NextX(t)
	if err != nil {
		return err
	}
	chart.mutex.Lock()
	defer chart.mutex.Unlock()
	candle := chart.candleTimeSeries[x]
	if candle == nil {
		if chart.currentCandle == nil {
			candle = NewCandle(x, decimal.Decimal{}, 0)
		} else {
			candle = NewCandle(x, chart.currentCandle.Close, 0)
		}
		chart.lastCandle = chart.currentCandle
		chart.currentCandle = candle
		chart.candleTimeSeries[x] = candle
		chart.candlesticks = append(chart.candlesticks, candle)
		if chart.watchFunc != nil {
			chart.watchFunc(candle)
		}
	}
	return nil
}

func (chart *CandlestickChart) AddTrade(t time.Time, value decimal.Decimal, volume int64) error {
	res := make([]*Candlestick, 0)
	x, err := chart.timeSeries.NextX(t)
	if err != nil {
		return err
	}
	if chart.currentCandle != nil && x.Before(chart.currentCandle.Time) {
		return errors.New("time before current candle")
	}
	chart.mutex.Lock()
	defer chart.mutex.Unlock()
	candle := chart.candleTimeSeries[x]
	if candle != nil {
		candle.Add(value, volume)
	} else {
		candle = NewCandle(x, value, volume)
		chart.lastCandle = chart.currentCandle
		chart.currentCandle = candle
		chart.candleTimeSeries[x] = candle
		chart.candlesticks = append(chart.candlesticks, candle)
	}
	if chart.watchFunc != nil {
		chart.watchFunc(candle)
	}
	res = append(res, candle)
	return nil
}

func (chart *CandlestickChart) AddVolume(t time.Time, volume int64) error {
	return chart.AddTrade(t, decimal.Decimal{}, volume)
}

func (chart *CandlestickChart) AppendTimeRange(start, end time.Time) error {
	return chart.timeSeries.AppendTimeRange(start, end)
}

func (chart *CandlestickChart) WatchFunc(f func(*Candlestick)) {
	chart.watchFunc = f
}

func (chart *CandlestickChart) Stop() {
	close(chart.exit)
}

type TimeSeries struct {
	period    Period
	timeRange map[time.Time]*DayTime
	endTime   time.Time
	loc       *time.Location
}

func (ts *TimeSeries) ToX(t time.Time) (time.Time, error) {
	if !ts.onRange(t) {
		return time.Time{}, errors.New("not on time range")
	}
	return ts.toX(t), nil
}

func (ts *TimeSeries) onRange(t time.Time) bool {
	dt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, ts.loc)
	dayTime, ok := ts.timeRange[dt]
	if !ok {
		return false
	}
	if t.Before(dayTime.Start) || t.After(dayTime.End) {
		return false
	}
	return true
}

func (ts *TimeSeries) toX(t time.Time) time.Time {
	t = t.In(ts.loc)
	switch ts.period {
	case PeriodMinute:
		return t.Truncate(time.Minute)
	case PeriodFiveMinute:
		return t.Truncate(5 * time.Minute)
	case PeriodQuarterHour:
		return t.Truncate(15 * time.Minute)
	case PeriodHalfHour:
		return t.Truncate(30 * time.Minute)
	case PeriodHour:
		return t.Truncate(time.Hour)
	case PeriodDay:
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, ts.loc)
	case PeriodWeek:
		offset := t.Weekday() - 1
		if offset < 0 {
			offset = 6
		}
		nt := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, ts.loc)
		nt = nt.Add(time.Duration(-offset) * 24 * time.Hour)
		return nt
	case PeriodMonth:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, ts.loc)
	case PeriodYear:
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, ts.loc)
	default:
		return t.Truncate(time.Minute)
	}
}

func (ts *TimeSeries) NextX(t time.Time) (time.Time, error) {
	x := ts.toX(t)
	switch ts.period {
	case PeriodMinute:
		x = x.Add(time.Minute)
	case PeriodFiveMinute:
		x = x.Add(5 * time.Minute)
	case PeriodQuarterHour:
		x = x.Add(15 * time.Minute)
	case PeriodHalfHour:
		x = x.Add(30 * time.Minute)
	case PeriodHour:
		x = x.Add(time.Hour)
	case PeriodDay:
		x = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, ts.loc)
	case PeriodWeek:
		x = time.Date(t.Year(), t.Month(), t.Day()+7, 0, 0, 0, 0, ts.loc)
	case PeriodMonth:
		x = time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, ts.loc)
	case PeriodYear:
		x = time.Date(t.Year()+1, 0, 0, 0, 0, 0, 0, ts.loc)
	default:
		x = t.Add(time.Minute)
	}

	dt := time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, ts.loc)
	dayTime, ok := ts.timeRange[dt]
	if ok && x.Before(dayTime.End) && x.After(dayTime.Start) {
		return x, nil
	} else if ok && x.Before(dayTime.Start) {
		return dayTime.Start, nil
	}
	for ; dt.Before(ts.endTime); dt.AddDate(0, 0, 1) {
		dayTime, ok = ts.timeRange[dt]
		if ok {
			return dayTime.Start, nil
		}
	}
	return time.Time{}, errors.New("no next x")
}

func (ts *TimeSeries) AppendTimeRange(start, end time.Time) error {
	dt := &DayTime{
		Start: start,
		End:   end,
	}
	t := dt.Start.Truncate(24 * time.Hour)
	ts.timeRange[t] = dt
	return nil
}
