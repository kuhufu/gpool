package gpool

import (
	"fmt"
	"testing"
	"time"
)

func TestPool_Run(t *testing.T) {
	p := New(4, 4, 6, ByChan)

	for i := 0; i < 20; i++ {
		p.Run(func() {
			fmt.Println("hello word")
		})
	}
	p.WaitAndClose()
	time.Sleep(time.Second)
}
