package main

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

const MessageDeadPlease Message = 666

type Worker struct {
	dead         *atomic.Bool
	deadCallback func()
	id           int
	ctx          context.Context
	isGood       bool
}

func NewWorker(ctx context.Context, id int, isGood bool, deadCallback func()) Worker {
	w := Worker{
		id:           id,
		ctx:          ctx,
		dead:         &atomic.Bool{},
		isGood:       isGood,
		deadCallback: deadCallback,
	}
	go w.run()
	return w
}

func (w Worker) isDead() bool {
	// хорошие воркеры проверяют сигналы, плохие - игнорируют
	return w.isGood && w.dead.Load()
}

func (w Worker) run() {
	log.Printf("worker-%d/%t: я родился", w.id, w.isGood)
	defer func() {
		log.Printf("worker-%d/%t: я умер", w.id, w.isGood)
		w.deadCallback()
	}()

	for {
		log.Printf("worker-%d/%t: жду (1сек)", w.id, w.isGood)
		time.Sleep(time.Second)
		if w.isDead() {
			return
		}

		log.Printf("worker-%d/%t: иду в базу с тяжелым запросом (3сек)", w.id, w.isGood)
		if w.isDead() {
			return
		}

		t := time.NewTimer(time.Second * 3)
		// решили сходить в базу или мкс
		select {
		case <-w.ctx.Done():
			log.Printf("worker-%d/%t: не смог сходить в базу - контекст протух", w.id, w.isGood)
			return
		case <-t.C:
			log.Printf("worker-%d/%t: таймер", w.id, w.isGood)
		}

		log.Printf("worker-%d/%t: сходил в базу", w.id, w.isGood)

		if w.isDead() {
			return
		}
	}
}

func (w Worker) Name() string {
	return fmt.Sprintf("worker-%d/%t", w.id, w.isGood)
}

func (w Worker) Handle(message Message) {
	if message == MessageDeadPlease {
		w.dead.Store(true)
	}
}
