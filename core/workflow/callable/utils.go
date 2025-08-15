// Package callable provides utility functions for workflow callable operations,
// including timeout handling and trigger expression parsing.
package callable

import (
	"strconv"
	"strings"
	"time"
)

func AcquireTimeout(defaultTimeout time.Duration, varStack map[string]string, callName string, envId string) time.Duration {
	timeout := defaultTimeout
	timeoutStr, ok := varStack["__call_timeout"] // the Call interface ensures we'll find this key
	// see Call.Call in callable/call.go for details
	if ok {
		var err error
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			timeout = defaultTimeout
			log.WithField("partition", envId).
				WithField("call", callName).
				WithField("default", timeout.String()).
				Warn("could not parse timeout declaration for hook call")
		}
	} else {
		log.WithField("partition", envId).
			WithField("call", callName).
			WithField("default", timeout.String()).
			Warn("could not get timeout declaration for hook call")
	}
	return timeout
}

func ParseTriggerExpression(triggerExpr string) (triggerName string, triggerWeight HookWeight) {
	var (
		triggerWeightS string
		triggerWeightI int
		err            error
	)

	// Split the trigger expression of this task by + or -
	if splitIndex := strings.LastIndexFunc(triggerExpr, func(r rune) bool {
		return r == '+' || r == '-'
	}); splitIndex >= 0 {
		triggerName, triggerWeightS = triggerExpr[:splitIndex], triggerExpr[splitIndex:]
	} else {
		triggerName, triggerWeightS = triggerExpr, "+0"
	}

	triggerWeightI, err = strconv.Atoi(triggerWeightS)
	if err != nil {
		log.Warnf("invalid trigger weight definition %s, defaulting to %s", triggerExpr, triggerName+"+0")
		triggerWeightI = 0
	}
	triggerWeight = HookWeight(triggerWeightI)

	return
}
