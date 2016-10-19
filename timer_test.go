package timer

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

var sum int32 = 0
var N int32 = 100
var tw *TimeWheel

func now() {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"))
	atomic.AddInt32(&sum, 1)
	v := atomic.LoadInt32(&sum)
	if v == 2*N {
		tw.Stop()
	}

}

func TestTimer(t *testing.T) {
	timerwheel := NewTimeWheel(time.Millisecond * 10)
	tw = timerwheel
	fmt.Println(timerwheel)
	var i int32
	for i = 0; i < N; i++ {
		timerwheel.AddNode(time.Millisecond*time.Duration(100*i), now)
		timerwheel.AddNode(time.Millisecond*time.Duration(100*i), now)
	}
	timerwheel.Start()
	if sum != 2*N {
		t.Error("failed")
	}
}
