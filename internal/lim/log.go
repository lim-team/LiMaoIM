package lim

import "github.com/tangtaoit/limnet/pkg/limlog"

type logger struct {
	log limlog.Log
}

func newLogger() *logger {
	return &logger{
		log: limlog.NewLIMLog("raft"),
	}
}

func (l *logger) Debug(v ...interface{}) {
}
func (l *logger) Debugf(format string, v ...interface{}) {

}

func (l *logger) Error(v ...interface{}) {

}
func (l *logger) Errorf(format string, v ...interface{}) {

}

func (l *logger) Info(v ...interface{}) {

}
func (l *logger) Infof(format string, v ...interface{}) {

}

func (l *logger) Warning(v ...interface{}) {

}
func (l *logger) Warningf(format string, v ...interface{}) {

}

func (l *logger) Fatal(v ...interface{}) {

}
func (l *logger) Fatalf(format string, v ...interface{}) {

}

func (l *logger) Panic(v ...interface{}) {

}
func (l *logger) Panicf(format string, v ...interface{}) {

}
