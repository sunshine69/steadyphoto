package utils

import (
	"github.com/jbrodriguez/mlog"
)

func init() {
	mlog.Start(mlog.LevelError, "")
}

// // Errorf formats according to a format specifier and logs it as an error
// func Errorf(format string, a ...interface{}) {
// 	mlog.Error(fmt.Errorf(format, a...))
// }

// // Error wraps a standard string into an error object automatically
// func Error(msg string) {
// 	mlog.Error(errors.New(msg))
// }
