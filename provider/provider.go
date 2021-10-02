package provider

import (
	"context"
	"github.com/pzierahn/project.go.omnetpp/eval"
	"github.com/pzierahn/project.go.omnetpp/gconfig"
	pb "github.com/pzierahn/project.go.omnetpp/proto"
	"github.com/pzierahn/project.go.omnetpp/simple"
	"github.com/pzierahn/project.go.omnetpp/stargate"
	"github.com/pzierahn/project.go.omnetpp/stargrpc"
	"github.com/pzierahn/project.go.omnetpp/storage"
	"github.com/pzierahn/project.go.omnetpp/sysinfo"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"
)

type simulationId = string

type provider struct {
	pb.UnimplementedProviderServer
	providerId     string
	numJobs        int
	store          *storage.Server
	slots          chan int
	mu             *sync.RWMutex
	sessions       map[simulationId]*pb.Session
	executionTimes map[simulationId]time.Duration
	newRecv        *sync.Cond
	allocRecvs     map[simulationId]chan<- int
}

func Start(config gconfig.Config) {

	mu := &sync.RWMutex{}
	prov := &provider{
		providerId:     simple.NamedId(config.Provider.Name, 8),
		numJobs:        config.Provider.Jobs,
		store:          &storage.Server{},
		slots:          make(chan int, config.Provider.Jobs),
		mu:             mu,
		newRecv:        sync.NewCond(mu),
		sessions:       make(map[simulationId]*pb.Session),
		executionTimes: make(map[simulationId]time.Duration),
		allocRecvs:     make(map[simulationId]chan<- int),
	}

	log.Printf("start provider (%v)", prov.providerId)

	//
	// Init stuff
	//

	prov.recoverSessions()

	simple.Watch("/sessions", func() interface{} {
		mu.RLock()
		defer mu.RUnlock()

		return prov.sessions
	})
	simple.Watch("/executionTimes", func() interface{} {
		mu.RLock()
		defer mu.RUnlock()

		data := make(map[string]string)

		for id, dur := range prov.executionTimes {
			data[id] = dur.String()
		}

		return data
	})

	go simple.StartWatchServer()

	//
	// Register provider
	//

	log.Printf("connect to broker %v", config.Broker.BrokerDialAddr())

	brokerConn, err := grpc.Dial(
		config.Broker.BrokerDialAddr(),
		grpc.WithInsecure(),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalln(err)
	}

	eval.DeviceId = prov.providerId
	eval.Init(brokerConn)

	broker := pb.NewBrokerClient(brokerConn)

	stream, err := broker.Register(context.Background())
	if err != nil {
		log.Fatalln(err)
	}

	err = stream.Send(&pb.Ping{Cast: &pb.Ping_Register{Register: prov.info()}})
	if err != nil {
		log.Fatalln(err)
	}

	go func() {
		for range time.Tick(time.Millisecond * 500) {

			var util *pb.Utilization
			util, err = sysinfo.GetUtilization(context.Background())
			if err != nil {
				log.Fatalln(err)
			}

			//log.Printf("Start: send utilization %v", util.CpuUsage)

			err = stream.Send(&pb.Ping{Cast: &pb.Ping_Util{Util: util}})
			if err != nil {
				// TODO: reconnect after EOF
				log.Fatalln(err)
			}
		}
	}()

	//
	// Start stargate-gRPC servers.
	//

	server := grpc.NewServer()
	pb.RegisterProviderServer(server, prov)
	pb.RegisterStorageServer(server, prov.store)

	stargate.SetConfig(stargate.Config{
		Addr: config.Broker.Address,
		Port: config.Broker.StargatePort,
	})

	go stargrpc.ServeLocal(prov.providerId, server)
	go stargrpc.ServeP2P(prov.providerId, server)
	go stargrpc.ServeRelay(prov.providerId, server)

	//
	// Start resource allocator.
	//

	prov.startAllocator(config.Provider.Jobs)

	return
}
