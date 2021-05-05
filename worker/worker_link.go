package worker

import (
	"context"
	"fmt"
	pb "github.com/patrickz98/project.go.omnetpp/proto"
	"google.golang.org/grpc/metadata"
	"runtime"
)

func (client *workerConnection) StartLink(ctx context.Context) (err error) {

	logger.Println("start worker", client.workerId)

	md := metadata.New(map[string]string{
		"workerId": client.workerId,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"numCPU":   fmt.Sprint(runtime.NumCPU()),
	})

	ctx = metadata.NewOutgoingContext(ctx, md)

	// Link to the work stream
	link, err := client.broker.TaskSubscription(ctx)
	if err != nil {
		return
	}
	defer func() { _ = link.CloseSend() }()

	exit := make(chan bool)
	defer close(exit)

	work := make(chan *pb.Task)
	defer close(work)
	go func() {

		//
		// Single thread to receive tasks
		//

		for {
			var task *pb.Task
			task, err = link.Recv()
			if err != nil {
				logger.Printf("work receiver: %v", err)
				break
			}

			logger.Printf("receive work %v_%v_%v", task.SimulationId, task.Config, task.RunNumber)
			work <- task
		}

		logger.Printf("exit work receiver")
	}()

	sendWorkReq := make(chan bool)
	defer close(sendWorkReq)
	go func() {
		for {
			send, ok := <-sendWorkReq
			if !ok {
				break
			}

			if !send {
				continue
			}

			err = client.SendWorkRequest(link)
			if err != nil {
				logger.Printf("send work request: %v", err)
				break
			}
		}

		logger.Printf("exit work request sender")
	}()

	for idx := 0; idx < client.agents; idx++ {

		//
		// Start worker agents
		//

		go func(idx int) {
			for {
				logger.Printf("agent %d send work request", idx)
				sendWorkReq <- true

				logger.Printf("agent %d waiting for work", idx)
				task, ok := <-work
				if !ok {
					break
				}

				logger.Printf("agent %d received work (%s_%s_%s)",
					idx, task.SimulationId, task.Config, task.RunNumber)

				client.runTasks(task)
			}

			logger.Printf("agent %d exiting", idx)
			exit <- true
		}(idx)
	}

	for idx := 0; idx < client.agents; idx++ {
		<-exit
	}

	logger.Println("closing connection to broker")

	return
}
