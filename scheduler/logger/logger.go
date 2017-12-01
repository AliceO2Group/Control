package logger

import (
	"github.com/sirupsen/logrus"
)

type Log struct {
	logrus.Entry
}

func (logger *Log) WithPrefix(prefix string) *logrus.Entry {
	return logger.WithField("prefix", prefix)
}

func New(baseLogger *logrus.Logger, defaultPrefix string) *Log {
	logger := new(Log)
	logger.Logger = baseLogger
	logger.Data = make(logrus.Fields, 5)
	logger.Data["prefix"] = defaultPrefix
	return logger
}
