package gpool

import (
	"fmt"
	"testing"
	"time"
)

func TestPool_Run(t *testing.T) {
	p := New(4, 4, 6, ByChan)
	p.Start()
	for i := 0; i < 10; i++ {
		p.Run(func() {
			fmt.Println("hello word")
		})

		if i == 5 {
			p.ShutdownDirectly()
		}
	}
	time.Sleep(time.Second)

	time.Sleep(time.Second)
}

func TestPool_ShutdownDirectly(t *testing.T) {
	p := New(4, 4, 6, ByChan)
	p.Start()
	for i := 0; i < 1000; i++ {
		p.Run(func() {
			fmt.Println("hello word")
		})

		if i == 500 {
			p.ShutdownDirectly()
		}
	}
	time.Sleep(time.Second)
}

func TestPool_ShutdownGracefully(t *testing.T) {
	p := New(4, 4, 6, ByChan)
	p.Start()
	for i := 0; i < 10; i++ {
		p.Run(func() {
			fmt.Println("hello word")
		})

		if i == 5 {
			p.ShutdownGracefully()
		}
	}
	time.Sleep(time.Second)
	time.Sleep(time.Second)
}

func TestPool_WaitAndClose(t *testing.T) {
	p := New(4, 4, 6, ByChan)
	p.Start()
	for i := 0; i < 1000; i++ {
		fmt.Println(<-p.Call(func() interface{} {
			return "haha"
		}).Done())
	}

	p.Wait()

	p.ShutdownDirectly()
	time.Sleep(time.Second)
}

func TestPool_Call(t *testing.T) {
	p := New(4, 4, 6, ByChan)

	res := p.Call(func() interface{} {
		fmt.Println("hello")
		return "world"
	})

	fmt.Println(<-res.Done())
}
