package logging // import "github.com/riking/DisGoBot/logging"

import "fmt"

func Debug(args ...interface{}) {
	fmt.Println(append([]interface{}{"[DBUG]"}, args...)...)
}
func Info(args ...interface{}) {
	fmt.Println(append([]interface{}{"[INFO]"}, args...)...)
}
func Warn(args ...interface{}) {
	fmt.Println(append([]interface{}{"[WARN]"}, args...)...)
}
func Error(args ...interface{}) {
	fmt.Println(append([]interface{}{"[EROR]"}, args...)...)
}
