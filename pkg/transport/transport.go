package transport

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

type Opt struct {
	Threads      uint
	Retries      uint
	RetriesDelay time.Duration
}

type Transport struct {
	opt         Opt
	queueLength int
	queue       chan Task
	complete    chan Task
	errors      chan Task
}

func NewTransport(opt Opt) *Transport {
	t := &Transport{
		opt:         opt,
		queueLength: 0,
		queue:       make(chan Task),
		complete:    make(chan Task),
		errors:      make(chan Task),
	}
	return t
}

func (t *Transport) Start() {
	for i := uint(0); i < t.opt.Threads; i++ {
		go t.startThread(i)
	}
	go t.listen()
}

func (t *Transport) Close() {
	close(t.queue)
}

func (t *Transport) AddTask(task Task) {
	t.queueLength++
	t.queue <- task
}

func (t *Transport) listen() {
	fmt.Println("Transport listening for complete tasks and errors..")
}

func (t *Transport) startThread(id uint) {
	fmt.Printf("[%d] Transport startThread\n", id)
	for {
		select {
		case task, success := <-t.queue:
			if !success {
				fmt.Printf("[%d] Transport queue (len %d), %#v\n", id, t.queueLength, task)
				return
				break
			}
			fmt.Printf("[%d] Transport queue (len %d), %#v\n", id, t.queueLength, task)

		}
	}
}

func (t *Transport) requestTry(reqFn func() (*http.Request, error)) (resp *http.Response, err error) {
	var req *http.Request
	for i := 0; i < c.RetryLimit; i++ {
		resp = nil
		req, err = reqFn()
		if err == nil {
			resp, err = c.httpClient.Do(req)
		}
		if err == nil && resp != nil && resp.StatusCode != 429 {
			return
		}
		log.Printf("request retry %d of %d: ", i+1, c.RetryLimit)
		time.Sleep(c.RetryDelay)
	}
	log.Println("request tried out", err)
	return
}

type Task struct {
}
