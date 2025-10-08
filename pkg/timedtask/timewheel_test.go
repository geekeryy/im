package timedtask

import (
	"log"
	"testing"
	"time"
)

func TestTimeWheel(t *testing.T) {
	timeWheel := NewTimeWheel(time.Second, 10)

	log.Println("add task")
	timeWheel.AddDelayTask(func() {
		log.Println("task")
	}, time.Second*5)

	timeWheel.AddIntervalTask(func() {
		log.Println("interval task")
	}, time.Second*3)

	timeWheel.AddIntervalTask(func() {
		log.Println("interval task13")
	}, time.Second*13)

	select {}
}
