package timedtask

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

type TimeWheel struct {
	interval time.Duration
	slots    []*Slot
	currSlot int
}

type Slot struct {
	head *Task
	tail *Task
}

type Task struct {
	tType    int // 1: delay, 2: interval
	circle   int
	period   time.Duration
	uuid     string
	function func()
	next     *Task
}

func (s *Slot) AddTask(task *Task) {
	if s.head == nil {
		s.head = task
		s.tail = task
	} else {
		s.tail.next = task
	}
}

func (s *Slot) RemoveTask(uuid string) {
	for preTask, task := s.head, s.head; task != nil; {
		if task.uuid == uuid {
			next := task.next
			task.next = nil
			if preTask != task {
				preTask.next = next
			}
			task = next
			break
		} else {
			preTask = task
			task = task.next
		}
	}
}

func NewTimeWheel(interval time.Duration, slotnum int) *TimeWheel {
	slots := make([]*Slot, slotnum)
	for i := 0; i < slotnum; i++ {
		slots[i] = &Slot{
			head: nil,
			tail: nil,
		}
	}
	t := &TimeWheel{
		interval: interval,
		slots:    slots,
	}
	go t.Start()
	return t
}

func (t *TimeWheel) Start() {
	ticker := time.NewTicker(t.interval)
	for range ticker.C {
		currSlot := t.currSlot

		fmt.Println("currSlot", currSlot)
		slot := t.slots[currSlot]
		for preTask, task := slot.head, slot.head; task != nil; {
			if task.circle > 0 {
				task.circle--
				preTask = task
				task = task.next
				continue
			}
			go task.function()

			switch task.tType {
			case 2:
				t.addIntervalTask(task.uuid, task.function, task.period)
			}

			next := task.next
			task.next = nil
			if preTask == task {
				slot.head = next
				preTask = next
			} else {
				preTask.next = next
			}

			task = next
		}

		t.currSlot = (currSlot + 1) % len(t.slots)
	}
}

// 延迟任务，在delay时间后执行
func (t *TimeWheel) AddDelayTask(function func(), delay time.Duration) string {
	uuid := uuid.New().String()
	circle := int(float64(delay) / float64(t.interval*time.Duration(len(t.slots))))
	pos := (t.currSlot + int(math.Ceil(float64(delay)/float64(t.interval)))) % len(t.slots)
	t.slots[pos].AddTask(&Task{
		uuid:     uuid,
		tType:    1,
		function: function,
		circle:   int(circle),
	})
	fmt.Println("addDelayTask", uuid, t.currSlot, pos, circle)
	return uuid
}

// 间隔任务，每interval时间执行一次
func (t *TimeWheel) AddIntervalTask(function func(), period time.Duration) string {
	uuid := uuid.New().String()
	t.addIntervalTask(uuid, function, period)
	return uuid
}

func (t *TimeWheel) addIntervalTask(uuid string, function func(), period time.Duration) string {
	circle := int(float64(period) / float64(t.interval*time.Duration(len(t.slots))))
	pos := (t.currSlot + int(math.Ceil(float64(period)/float64(t.interval)))) % len(t.slots)
	fmt.Println("addIntervalTask", uuid, t.currSlot, pos, circle)
	t.slots[pos].AddTask(&Task{
		uuid:     uuid,
		tType:    2,
		function: function,
		circle:   int(circle),
		period:   period,
	})
	return uuid
}

// 移除任务
func (t *TimeWheel) RemoveTask(uuid string) {
	for _, slot := range t.slots {
		slot.RemoveTask(uuid)
		break
	}
}
