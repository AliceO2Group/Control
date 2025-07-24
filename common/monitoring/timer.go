package monitoring

import (
	"time"

	"github.com/AliceO2Group/Control/common/logger/infologger"
)

type TimeResolution int

const (
	Millisecond TimeResolution = iota
	Nanosecond
)

// Timer function is meant to be used with defer statement to measure runtime of given function:
// defer Timer(&metric, Milliseconds)()
func Timer(metric *Metric, unit TimeResolution) func() {
	return timer(metric, unit, false, false)
}

// Timer function is meant to be used with defer statement to measure runtime of given function:
// defer Timer(&metric, Milliseconds)()
// sends measured value as Send(metric)
func TimerSendSingle(metric *Metric, unit TimeResolution) func() {
	return timer(metric, unit, true, false)
}

// Timer function is meant to be used with defer statement to measure runtime of given function:
// defer Timer(&metric, Milliseconds)()
// sends measured value as SendHistogrammable(metric)
func TimerSendHist(metric *Metric, unit TimeResolution) func() {
	return timer(metric, unit, true, true)
}

func timer(metric *Metric, unit TimeResolution, send bool, sendHistogrammable bool) func() {
	start := time.Now()

	return func() {
		dur := time.Since(start)
		// we are setting default value as Nanoseconds
		switch unit {
		case Millisecond:
			metric.SetFieldInt64("execution_time_ms", dur.Milliseconds())
		case Nanosecond:
			metric.SetFieldInt64("execution_time_ns", dur.Nanoseconds())
		default:
			log.WithField("level", infologger.IL_Devel).Warnf("trying to use unknown time resolution in monitoring.timer function [%d], skipping", unit)
		}

		if send {
			if sendHistogrammable {
				SendHistogrammable(metric)
			} else {
				Send(metric)
			}
		}
	}
}
