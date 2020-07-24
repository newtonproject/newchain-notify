package notify

import (
	log "github.com/sirupsen/logrus"
)

type errorLogger struct {
	*log.Logger
}

func (e errorLogger) Println(v ...interface{}) {
	e.Errorln(v)
}

func (e errorLogger) Printf(format string, v ...interface{}) {
	e.Errorf(format, v...)
}

type warnLogger struct {
	*log.Logger
}

func (e warnLogger) Println(v ...interface{}) {
	e.Warnln(v)
}

func (e warnLogger) Printf(format string, v ...interface{}) {
	e.Warnf(format, v...)
}

type debugLogger struct {
	*log.Logger
}

func (e debugLogger) Println(v ...interface{}) {
	e.Debugln(v)
}

func (e debugLogger) Printf(format string, v ...interface{}) {
	e.Debugf(format, v...)
}
