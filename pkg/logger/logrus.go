package logger

import (
	"github.com/sirupsen/logrus"
)

type LogrusLogger struct {
	*logrus.Entry
}

func (log *LogrusLogger) SetField(key string, value interface{}) {
	log.WithField(key, value)
}
