package main

import (
	proto "Consensus/grpc"
	"sync"

	"google.golang.org/grpc"
	//"google.golang.org/grpc/credentials/insecure"
)

// Make nodes work as both client AND server #lifehack

const (
	held     string = "held"
	released string = "released"
	wanted   string = "wanted"
)

type Node struct {
	proto.UnimplementedMutexServiceServer
	state   string
	nodeID  int32
	replies chan int
	mutex   sync.Mutex
	// noget med channels
}

type Connection struct {
	node           proto.MutexServiceClient
	nodeConnection *grpc.ClientConn
}

func main() {

	node := Node{
		state: "released",
	}

}
