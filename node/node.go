package main

import (
	proto "Consensus/grpc"
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Måske først start alle noder og SÅ forbind dem #genius

// Make nodes work as both client AND server #lifehack
// go run node.go -port=5001 -name=1 -peers="127.0.0.1:5002,127.0.0.1:5003"
// go run node.go -port=5002 -name=2 -peers="127.0.0.1:5001,127.0.0.1:5003"
// go run node.go -port=5003 -name=3 -peers="127.0.0.1:5001,127.0.0.1:5002"

const (
	held      string = "held"
	released  string = "released"
	requested string = "requested"
)

type Node struct {
	proto.UnimplementedMutexServiceServer
	state          string
	nodeID         int32
	replies        chan int
	mutex          sync.Mutex
	lamport        int32
	queue          []int32
	connections    map[int32]*Connection
	pendingReplies int
}

type Connection struct {
	node           proto.MutexServiceClient
	nodeConnection *grpc.ClientConn
}

var port = flag.String("port", "5400", "listening at port")
var name = flag.Int("name", 0, "the node's name")
var peerAddresses = flag.String("peers", "", "Comma-separated list of other node addresses in the format ip:port")

func main() {

	// parsing the CLI input
	flag.Parse()

	// writing to the log
	file, err := os.OpenFile("../nodeLog.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer file.Close()

	// Writing das file, jaaa
	log.SetOutput(file)

	log.Println("---------- Starting Node ----------")

	node := Node{
		state:          released,
		nodeID:         int32(*name),
		lamport:        1,
		replies:        make(chan int),
		connections:    make(map[int32]*Connection),
		pendingReplies: 0,
	}

	go node.instansiateNode()
	node.connectNodes()

	go node.handleInput()

	for {
		time.Sleep(5 * time.Second)
	}
	log.Printf("Node %d finished running main", node.nodeID)
}

func (node *Node) instansiateNode() {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", *port))
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", *port, err)
		return
	}

	grpcServer := grpc.NewServer()
	proto.RegisterMutexServiceServer(grpcServer, node)

	log.Printf("Node %d listening on port %s\n", *name, *port)
	fmt.Printf("Node %d listening on port %s\n", *name, *port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve %v", err)
	}

	log.Printf("Node %d finished running instansiateNode", node.nodeID)
}

func (node *Node) connectNodes() {
	peers := *peerAddresses
	if peers == "" {
		fmt.Println("No peers specified, running in standalone mode")
		return
	}

	peerList := strings.Split(peers, ",")
	for index, peerAddr := range peerList {

		nodeID := int32(index + 1)

		log.Printf("Connecting to peer with nodeID: %d, address: %s\n", nodeID, peerAddr)

		var conn *grpc.ClientConn
		var err error
		for retries := 0; retries < 5; retries++ {
			conn, err = grpc.Dial(peerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err == nil {
				break
			}

			log.Printf("Attempt %d to connect to peer %s failed: %v", retries+1, peerAddr, err)
			time.Sleep(time.Second)
		}

		if err != nil {
			log.Printf("Failed to connect to peer %s after 5 attempts: %v", peerAddr, err)
			continue
		}

		client := proto.NewMutexServiceClient(conn)

		//nodeID := int32(strings.Split(peerAddr, ":")[1][len(peerAddr)-1])
		node.connections[int32(nodeID)] = &Connection{
			node:           client,
			nodeConnection: conn,
		}

		//node.connections[nodeID] = &Connection{node: client, nodeConnection: conn}
		log.Printf("Connected node %d to peer %s", nodeID, peerAddr)
	}
}

func (node *Node) handleInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("**** type 'request' to enter the critical section ****\n")

	// To infinity, AND BEYOND
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if strings.Contains(input, "request") {
			go node.requestAccess()
			log.Printf("Node %d finished running requestAccess", node.nodeID)
		} else {
			fmt.Print("Not a known command, please try again :))\n")
		}
	}
}

func (node *Node) requestAccess() {
	node.incrementLamport()
	log.Printf("Node %d state change from %s to %s", node.nodeID, node.state, requested)
	node.state = requested
	node.pendingReplies = int(len(node.connections))

	log.Printf("Node %d is requesting access at lamport time %d", node.nodeID, node.lamport)

	for id, conn := range node.connections {
		_, err := conn.node.RequestMessage(context.Background(), &proto.Request{
			NodeId:  node.nodeID,
			Lamport: node.lamport,
			Port:    *port,
		})
		if err != nil {
			log.Printf("Error requesting access from node %d: %v", id, err)
		}
	}

	for node.pendingReplies > 0 {
		reply := <-node.replies
		node.pendingReplies--
		log.Printf("Node %d received reply (%v), pendingReplies: %d", node.nodeID, reply, node.pendingReplies)
	}

	node.enterCriticalSection()
	log.Printf("Node %d finished running requestAccess", node.nodeID)
}

func (node *Node) RequestMessage(ctx context.Context, req *proto.Request) (*proto.Reply, error) {

	node.incrementLamport()
	node.lamport = max(node.lamport, req.Lamport) + 1

	if node.state == held || (node.state == requested &&
		(node.lamport < req.Lamport || (node.lamport == req.Lamport && node.nodeID < req.NodeId))) {
		node.queue = append(node.queue, req.NodeId)
		log.Printf("Request from node %d deferred at lamport: %d", node.nodeID, node.lamport)

		return &proto.Reply{Message: "Request deferred", Lamport: node.lamport}, nil
	}

	log.Printf("Request from node %d granted at lamport: %d", node.nodeID, node.lamport)
	return &proto.Reply{Message: "Request granted", Lamport: node.lamport}, nil
}

func (node *Node) enterCriticalSection() {
	log.Printf("Node %d is running enterCriticalSection\n", node.nodeID)
	node.mutex.Lock()
	log.Printf("Node %d state change from %s to %s", node.nodeID, node.state, held)
	node.state = held
	node.mutex.Unlock()

	log.Printf("Node %d is entering the critical section\n", node.nodeID)

	time.Sleep(5 * time.Second)

	log.Printf("Node %d is leaving the critical section\n", node.nodeID)
	log.Printf("Node %d state change from %s to %s", node.nodeID, node.state, released)
	node.state = released

	node.sendDeferredReplies()

	log.Printf("Node %d finished running enterCriticalSection", node.nodeID)
}

func (node *Node) sendDeferredReplies() {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	for _, id := range node.queue {
		if conn, exists := node.connections[id]; exists {
			node.incrementLamport()
			log.Printf("Node %d sending deferred reply to Node %d at Lamport time %d", node.nodeID, id, node.lamport)
			_, err := conn.node.ReplyMessage(context.Background(), &proto.Reply{
				Message: "Request granted",
				Lamport: node.lamport,
			})
			if err != nil {
				log.Printf("Error sending deferred reply to node %d: %v", id, err)
			}
			log.Printf("Node %d finished running sendDeferredReplies", node.nodeID)
		}
	}

	// Clearing the queue folks
	node.queue = []int32{}
	log.Printf("Node %d finished running sendDeferredReplies (cleared the queue)", node.nodeID)
}

func (node *Node) ReplyMessage(ctx context.Context, req *proto.Reply) (*proto.Empty, error) {
	node.incrementLamport()
	node.lamport = max(node.lamport, req.Lamport) + 1

	node.replies <- 1

	return &proto.Empty{}, nil
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (node *Node) incrementLamport() {
	node.mutex.Lock()
	defer node.mutex.Unlock()
	node.lamport++
}
