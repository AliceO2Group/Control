package core

import (
	"time"

	xmetrics "github.com/mesos/mesos-go/api/v1/lib/extras/metrics"
	"github.com/sirupsen/logrus"
)

func forever(name string, jobRestartDelay time.Duration, counter xmetrics.Counter, f func() error) {
	for {
		counter(name)
		err := f()
		if err != nil {
			log.WithFields(logrus.Fields{
				"name": name,
				"error": err.Error(),
			}).Error("job exited with error")
		} else {
			log.WithField("name", name).Info("job exited")
		}
		time.Sleep(jobRestartDelay)
	}
}
