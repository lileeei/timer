package timer

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)


const (
	TIME_NEAR_SHIFT  = 8
	TIME_NEAR        = 1 << TIME_NEAR_SHIFT
	TIME_LEVEL_SHIFT = 6
	TIME_LEVEL       = 1 << TIME_LEVEL_SHIFT
	TIME_NEAR_MASK   = TIME_NEAR - 1
	TIME_LEVEL_MASK  = TIME_LEVEL - 1
)

type TimerWheel struct {
	near [TIME_NEAR]*list.List
	t    [4][TIME_LEVEL]*list.List
	sync.Mutex
	time uint32	     //时间轮time
	tick time.Duration   //时间轮的tick
	quit chan struct{}   //时间轮退出信号
}

type Node struct {
	//TimerId uint32  //定时器id，便于以后查找
	expire uint32	  //任务过期时间
	callBackFunc      func()  //回调函数
}

func (n *Node) String() string {
	return fmt.Sprintf("Node:expire,%d", n.expire)
}


//创建时间轮
func NewTimerWheel(d time.Duration) *TimerWheel {
	t := new(TimerWheel)
	t.time = 0
	t.tick = d
	t.quit = make(chan struct{})

	var i, j int
	for i = 0; i < TIME_NEAR; i++ {
		t.near[i] = list.New()
	}

	for i = 0; i < 4; i++ {
		for j = 0; j < TIME_LEVEL; j++ {
			t.t[i][j] = list.New()
		}
	}

	return t
}

func (tw *TimerWheel) AddNode(d time.Duration, f func()) *Node {
	n := new(Node)
	n.callBackFunc = f
	tw.Lock()
	n.expire = uint32(d/tw.tick) + tw.time
	tw.addNode(n)
	tw.Unlock()
	return n
}

func (tw *TimerWheel) addNode(n *Node) {
	expire := n.expire
	current := tw.time
	if (expire | TIME_NEAR_MASK) == (current | TIME_NEAR_MASK) {
		tw.near[expire&TIME_NEAR_MASK].PushBack(n)
	} else {
		var i uint32
		var mask uint32 = TIME_NEAR << TIME_LEVEL_SHIFT
		for i = 0; i < 3; i++ {
			if (expire | (mask - 1)) == (current | (mask - 1)) {
				break
			}
			mask <<= TIME_LEVEL_SHIFT
		}

		tw.t[i][(expire>>(TIME_NEAR_SHIFT+i*TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK].PushBack(n)
	}

}

func (tw *TimerWheel) String() string {
	return fmt.Sprintf("Timer:time:%d, tick:%s", tw.time, tw.tick)
}


//将timeout的任务从时间轮中删除
func dispatchList(front *list.Element) {
	for e := front; e != nil; e = e.Next() {
		node := e.Value.(*Node)
		go node.callBackFunc()
	}
}

func (tw *TimerWheel) moveList(level, idx int) {
	vec := tw.t[level][idx]
	front := vec.Front()
	vec.Init()
	for e := front; e != nil; e = e.Next() {
		node := e.Value.(*Node)
		tw.addNode(node)
	}
}

//转动时间轮
func (tw *TimerWheel) shift() {
	tw.Lock()
	var mask uint32 = TIME_NEAR
	tw.time++
	ct := tw.time
	
	if ct == 0 {
		tw.moveList(3, 0)
	} else {
		time := ct >> TIME_NEAR_SHIFT
		var i int = 0
		for (ct & (mask - 1)) == 0 {
			idx := int(time & TIME_LEVEL_MASK)
			if idx != 0 {
				tw.moveList(i, idx)
				break
			}
			mask <<= TIME_LEVEL_SHIFT
			time >>= TIME_LEVEL_SHIFT
			i++
		}
	}
	tw.Unlock()
}


//执行timeout的任务，并把它们从时间轮中删除，每次都从near中删除
func (tw *TimerWheel) execute() {
	tw.Lock()
	idx := tw.time & TIME_NEAR_MASK
	vec := tw.near[idx]
	if vec.Len() > 0 {
		front := vec.Front()
		vec.Init()
		tw.Unlock()

		dispatchList(front)
		return
	}

	tw.Unlock()
}

//更新时间轮
func (tw *TimerWheel) update() {
	tw.execute()
	tw.shift()
	tw.execute()
}


//开启时间轮
func (tw *TimerWheel) Start() {
	tick := time.NewTicker(tw.tick)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			tw.update()
		case <-tw.quit:
			return
		}
	}
}

func (tw *TimerWheel) Stop() {
	close(tw.quit)
}
