package monitoring

import (
	"time"
)

type TimeUnit int

const (
	Milliseconds TimeUnit = iota
	Nanoseconds
)

// Timer function is meant to be used with defer statement to measure runtime of given function:
// defer Timer(&metric, Milliseconds)()
func Timer(metric *Metric, unit TimeUnit) func() {
	return timer(metric, unit, false)
}

// TimerSend function is meant to be used with defer statement to measure runtime of given function:
// defer TimerSend(&metric, Milliseconds)()
func TimerSend(metric *Metric, unit TimeUnit) func() {
	return timer(metric, unit, true)
}

func timer(metric *Metric, unit TimeUnit, send bool) func() {
	start := time.Now()

	return func() {
		dur := time.Since(start)
		// we are setting default value as Nanoseconds
		if unit == Milliseconds {
			metric.SetFieldInt64("execution_time_ms", dur.Milliseconds())
		} else {
			metric.SetFieldInt64("execution_time_ns", dur.Nanoseconds())
		}

		if send {
			SendHistogrammable(metric)
		}
	}
}
