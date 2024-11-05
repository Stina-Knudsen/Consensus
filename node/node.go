package main

import (
	proto "Consensus/grpc"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	//"google.golang.org/grpc/credentials/insecure"
)

// Make nodes work as both client AND server #lifehack

const (
	held      string = "held"
	released  string = "released"
	requested string = "requested"
)

type Node struct {
	proto.UnimplementedMutexServiceServer
	state       string
	nodeID      int32
	replies     chan int
	mutex       sync.Mutex
	lamport     int32
	queue       []int32
	connections map[int32]*Connection
}

type Connection struct {
	node           proto.MutexServiceClient
	nodeConnection *grpc.ClientConn
}

var port = flag.String("port", "5400", "listening at port")
var name = flag.Int("name", 0, "the node's name")
var peerAddresses = flag.String("peers", "", "Comma-separated list of other node addresses in the format ip:port")

func main() {

	flag.Parse()
	// Now the real fun begins

	// Start the server/conection and such

	node := Node{
		state:       released,
		nodeID:      int32(*name),
		lamport:     1,
		replies:     make(chan int),
		connections: make(map[int32]*Connection),
	}

	go node.instansiateNode()

	node.connectNodes()

}

func (node *Node) instansiateNode() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", *port, err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterMutexServiceServer(grpcServer, node)

	fmt.Printf("Node %d listening on port %s\n", *name, *port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
}

func updateState() {

}

func (node *Node) connectNodes() {
	peers := *peerAddresses
	if peers == "" {
		fmt.Println("No peers specified, running in standalone mode")
		return
	}

	peerList := strings.Split(peers, ",")
	for _, peerAddr := range peerList {
		conn, err := grpc.Dial(peerAddr, grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect to peer %s: %v", peerAddr, err)
			continue
		}

		client := proto.NewMutexServiceClient(conn)
		nodeID := int32(strings.Split(peerAddr, ":")[1][len(peerAddr)-1]) // This is a placeholder; adjust based on node ID scheme
		node.connections[nodeID] = &Connection{node: client, nodeConnection: conn}

		log.Printf("Connected to peer %s", peerAddr)
	}
}

func (node *Node) requestAccess() {
	node.incrementLamport()
	node.state = requested

	// Request access

	// Vent på svar

	// Hop ind
}

func (node *Node) receiveReply() {

}

func (node *Node) enterCriticalSection() {
	node.mutex.Lock()
	node.state = held
	fmt.Printf("Node %d is entering the critical section\n", node.nodeID) //til loggen
	time.Sleep(2 * time.Second)
	fmt.Printf("Node %d is leaving the critical section\n", node.nodeID) //til loggen
	node.state = released
	node.mutex.Unlock()
}

func (node *Node) sendReply() {

}

func (node *Node) incrementLamport() {
	node.mutex.Lock()
	node.lamport++
	node.mutex.Unlock()
}

func checkLamport() {
	//if(node.lamport < )
}
