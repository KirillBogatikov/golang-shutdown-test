package main

import "log"

type Message int

type Consumer interface {
	Name() string
	Handle(Message)
}

type Broker interface {
	Subscribe(Consumer)
	Broadcast(Message)
}

type SimpleBroker struct {
	consumers    []Consumer
	newConsumers chan Consumer
	messages     chan Message
}

func NewSimpleBroker() Broker {
	s := SimpleBroker{
		newConsumers: make(chan Consumer, 1),
		messages:     make(chan Message, 128),
	}
	go s.listen()
	return s
}

func (s SimpleBroker) listen() {
	for {
		select {
		case c := <-s.newConsumers:
			s.consumers = append(s.consumers, c)
		case m := <-s.messages:
			for _, c := range s.consumers {
				func(c Consumer, m Message) {
					defer func() {
						if err := recover(); err != nil {
							log.Printf("consumer %s panic: %v", c.Name(), err)
						}
					}()

					c.Handle(m)
				}(c, m)
			}
		default:

		}
	}
}

func (s SimpleBroker) Subscribe(c Consumer) {
	s.newConsumers <- c
}

func (s SimpleBroker) Broadcast(m Message) {
	s.messages <- m
}
