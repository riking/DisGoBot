package logging // import "github.com/riking/DisGoBot/logging"

import "fmt"
import "time"

const DebugEnabled = true

func Debug(args ...interface{}) {
	if DebugEnabled {
		fmt.Println(append([]interface{}{fmt.Sprintf("[DBUG %s]", time.Now().Format(time.Kitchen))}, args...)...)
	}
}
func Info(args ...interface{}) {
	fmt.Println(append([]interface{}{fmt.Sprintf("[INFO %s]", time.Now().Format(time.Kitchen))}, args...)...)
}
func Warn(args ...interface{}) {
	fmt.Println(append([]interface{}{fmt.Sprintf("[WARN %s]", time.Now().Format(time.Kitchen))}, args...)...)
}
func Error(args ...interface{}) {
	fmt.Println(append([]interface{}{fmt.Sprintf("[EROR %s]", time.Now().Format(time.Kitchen))}, args...)...)
}
