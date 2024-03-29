package main

import (
	"context"
	"fmt"
	"github.com/frifox/fifo"
	"log/slog"
	"net/http"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var AppCtx context.Context
var AppTasks *sync.WaitGroup
var MyQueue *fifo.Queue[string, any, any]

func init() {
	AppCtx, _ = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	AppTasks = &sync.WaitGroup{}

	MyQueue = fifo.NewQueue[string, any, any](AppCtx)

	worker := Worker{
		AppCtx: AppCtx,
		Queue:  MyQueue,

		client:             http.Client{},
		fastRequestTimeout: time.Second * 1,
		slowRequestTimeout: time.Second * 10,
	}
	go worker.Run()

	slog.Info("init() complete")
}

func main() {
	reqA := WorkerRequestA{Email: "fastRequest@yahoo.com"}
	reqB := WorkerRequestB{Email: "slowRequest@gmail.com"}

	go pushJob(reqA.Email, reqA)
	go pushJob(reqA.Email, reqA) // duplicate
	go pushJob(reqB.Email, reqB)
	go pushJob("foo-bar", "foo") // unsupported request

	slog.Info("wait for job responses or hit CTRL+C")
	<-AppCtx.Done()

	slog.Info("AppCtx canceled, waiting for tasks to complete")
	AppTasks.Wait()

	slog.Info("app finished")
}

func pushJob(jobID string, request any) {
	AppTasks.Add(1)

	log := slog.With("jobID", jobID)
	log.Info("pushing job to queue")

	closure := func(response any) {
		switch response.(type) {
		case WorkerResponse:
			log.Info("job response received. Caching to database", "response", response)
		case error:
			log.Error("job response was an error", "err", response)
		default:
			log.Error("unsupported job response", "type", fmt.Sprintf("%T", response))
		}

		AppTasks.Done()
	}

	ok := MyQueue.AddAndCloseOnce(jobID, request, closure)
	if !ok {
		AppTasks.Done()
		log.Warn("job not added, already is in queue")
	}
}
