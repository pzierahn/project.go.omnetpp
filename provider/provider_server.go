package provider

import (
	"context"
	"github.com/lucas-clemente/quic-go"
	pnet "github.com/pzierahn/project.go.omnetpp/adapter"
	"github.com/pzierahn/project.go.omnetpp/gconfig"
	pb "github.com/pzierahn/project.go.omnetpp/proto"
	"github.com/pzierahn/project.go.omnetpp/simple"
	"github.com/pzierahn/project.go.omnetpp/stargate"
	"github.com/pzierahn/project.go.omnetpp/sysinfo"
	"github.com/pzierahn/project.go.omnetpp/utils"
	"google.golang.org/grpc"
	"log"
	"runtime"
	"time"
)

func Start(conf gconfig.Config) {

	prov := provider{
		providerId: simple.NamedId(runtime.GOOS+"-"+runtime.GOARCH, 8),
	}

	log.Printf("start provider (%v)", prov.providerId)

	//
	// Register provider
	//

	qconn, dialer := utils.GRPCDialerAuto()
	log.Printf("quic listener on %v", qconn.LocalAddr())

	brokerConn, err := grpc.Dial(
		conf.Broker.DialAddr(),
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithContextDialer(dialer),
	)
	if err != nil {
		log.Fatalln(err)
	}

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
			util, err = sysinfo.GetUtilization()
			if err != nil {
				log.Fatalln(err)
			}

			//log.Printf("Start: send utilization %v", util.CpuUsage)

			err = stream.Send(&pb.Ping{Cast: &pb.Ping_Util{Util: util}})
			if err != nil {
				log.Fatalln(err)
			}
		}
	}()

	//
	// Start provider
	//

	for {
		log.Println("wait for stargate connection")

		conn, _, err := stargate.Dial(context.Background(), prov.providerId)
		if err != nil {
			log.Fatalln(err)
		}

		go func() {
			defer func() { _ = conn.Close() }()

			tlsConf, _ := utils.GenerateTLSConfig()
			ql, err := quic.Listen(conn, tlsConf, nil)
			if err != nil {
				log.Fatalln(err)
			}

			log.Println("create adapter listener")
			lis := pnet.Listen(ql)
			defer func() { _ = lis.Close() }()

			log.Println("listening for consumer")

			server := grpc.NewServer()
			pb.RegisterProviderServer(server, &prov)
			err = server.Serve(lis)
		}()
	}

	return
}
