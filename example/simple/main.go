package main

import (
	"context"
	"github.com/frifox/fifo"
	"log/slog"
	"sync"
	"time"
)

func main() {
	queue := fifo.NewQueue[string, string, string](context.Background())

	tasks := sync.WaitGroup{}

	// job
	jobID := "myJobID"
	request := "some-data"
	closure := func(response string) {
		slog.Info("closure: request finished", "response", response)
		tasks.Done()
	}

	// launch 10 workers
	for i := 0; i < 10; i++ {
		go func() {
			for job := range queue.Jobs {
				slog.Info("worker: request received", "jobID", job.ID)

				time.Sleep(time.Second)
				response := job.Request + "=processed"
				time.Sleep(time.Second)

				slog.Info("worker: request proceessed", "jobID", job.ID)
				queue.Finish(job.ID, response)
			}
		}()
	}

	// queue job 10 times
	for i := 0; i < 10; i++ {
		tasks.Add(1)

		queued := queue.Add(jobID, request, closure)
		if queued {
			slog.Info("queue: job added")
		} else {
			slog.Warn("queue: job is already in the queue")
		}
	}

	// wait for all closures to execute
	tasks.Wait()
}
