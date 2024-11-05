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
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Make nodes work as both client AND server #lifehack

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
	pendingReplies int32
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
}

func (node *Node) connectNodes() {
	peers := *peerAddresses
	if peers == "" {
		fmt.Println("No peers specified, running in standalone mode")
		return
	}

	peerList := strings.Split(peers, ",")
	for _, peerAddr := range peerList {

		parts := strings.Split(peerAddr, ":")
		if len(parts) != 2 {
			log.Printf("Invalid peer format for %s, skipping", peerAddr)
			continue
		}

		// extract nodeID and port
		nodeIDStr := parts[0]
		port := parts[1]

		log.Printf("------- nodeID: %s\n", nodeIDStr)
		log.Printf("------- port: %s\n", port)

		// nodeID from string to integer, does not need
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			log.Printf("Invalid node ID for peer %s: %v", peerAddr, err)
			continue
		}

		address := fmt.Sprintf("%s:%s", parts[0], port)

		conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to connect to peer node %s on port %s: %v", nodeIDStr, port, err)
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
		} else {
			fmt.Print("Not a known command, please try again :))\n")
		}
	}
}

func (node *Node) requestAccess() {
	node.incrementLamport()
	node.pendingReplies = int32(len(node.connections))

	node.state = requested

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

	for i := 0; i < len(node.connections); i++ {
		<-node.replies
	}

	node.enterCriticalSection()
}

func (node *Node) RequestMessage(ctx context.Context, req *proto.Request) (*proto.Reply, error) {

	log.Printf("Node %d is requesting access at lamport time %d", node.nodeID, node.lamport)

	node.incrementLamport()
	node.lamport = max(node.lamport, req.Lamport) + 1

	if node.state == held || (node.state == requested && (node.lamport < req.Lamport || (node.lamport == req.Lamport && node.nodeID < req.NodeId))) {
		node.queue = append(node.queue, req.NodeId)
		log.Printf("Request from node %d deferred at lamport: %d", node.nodeID, node.lamport)
		return &proto.Reply{Message: "Request deferred", Lamport: node.lamport}, nil
	}

	log.Printf("Request from node %d granted at lamport: %d", node.nodeID, node.lamport)
	return &proto.Reply{Message: "Request granted", Lamport: node.lamport}, nil
}

func (node *Node) enterCriticalSection() {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	node.state = held
	log.Printf("Node %d is entering the critical section\n", node.nodeID)

	time.Sleep(5 * time.Second)

	log.Printf("Node %d is leaving the critical section\n", node.nodeID)
	node.state = released

	node.sendDeferredReplies()
}

func (node *Node) sendDeferredReplies() {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	for _, id := range node.queue {
		if conn, exists := node.connections[id]; exists {
			node.incrementLamport()
			_, err := conn.node.ReplyMessage(context.Background(), &proto.Reply{
				Message: "Request granted",
				Lamport: node.lamport,
			})
			if err != nil {
				log.Printf("Error sending deferred reply to node %d: %v", id, err)
			}
		}
	}

	// Clearing the queue folks
	node.queue = []int32{}
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
	node.lamport++
	node.mutex.Unlock()
}
