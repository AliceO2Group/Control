package monitoring

import "time"

// Timer* functions are meant to be used with defer statement to measure runtime of given function:
// defer TimerNS(&metric)()
func TimerMS(metric *Metric) func() {
	start := time.Now()
	return func() {
		metric.SetFieldInt64("execution_time_ms", time.Since(start).Milliseconds())
	}
}

func TimerNS(metric *Metric) func() {
	start := time.Now()
	return func() {
		metric.SetFieldInt64("execution_time_ns", time.Since(start).Nanoseconds())
	}
}
