package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	pb "Consensus/GRPC"

	"google.golang.org/grpc"
)

// Node represents a node in the token ring system
type Node struct {
	pb.UnimplementedTokenRingServiceServer
	name         string
	port         string
	nextNodeIP   string
	nextNodePort string
	hasToken     bool
}

var nodeQueue []string

// NewNode creates a new node instance
func NewNode(name, port, nextNodeIP, nextNodePort string, hasToken bool) *Node {
	return &Node{
		name:         name,
		port:         port,
		nextNodeIP:   nextNodeIP,
		nextNodePort: nextNodePort,
		hasToken:     true,
	}
}

// RequestToken handles a token request from another node
func (n *Node) RequestToken(ctx context.Context, req *pb.TokenRequest) (*pb.TokenResponse, error) {
	if n.hasToken {
		log.Printf("%s granting token to %s", n.name, req.GetNodeName())
		n.hasToken = false
		n.PassTokenToNext(req.GetNodeName())
		return &pb.TokenResponse{
			Status:  "granted",
			Message: fmt.Sprintf("Token passed to %s", req.GetNodeName()),
		}, nil
	}
	log.Printf("%s does not have the token to grant", n.name)
	return &pb.TokenResponse{
		Status:  "denied",
		Message: "Token not available",
	}, nil
}

// PassToken handles passing the token to the next node
func (n *Node) PassToken(ctx context.Context, token *pb.Token) (*pb.TokenResponse, error) {
	log.Printf("%s received the token from %s", n.name, token.GetHolder())
	n.hasToken = true
	return &pb.TokenResponse{
		Status:  "received",
		Message: "Token received",
	}, nil
}

// PassTokenToNext passes the token to the next node in the ring
func (n *Node) PassTokenToNext(holder string) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 5; i++ { // Retry up to 5 times
		conn, err = grpc.Dial(fmt.Sprintf("%s:%s", n.nextNodeIP, n.nextNodePort), grpc.WithInsecure())
		if err == nil {
			break
		}
		log.Printf("Failed to connect to next node, retrying... (%d/5)", i+1)
		time.Sleep(2 * time.Second) // Wait before retrying
	}
	if err != nil {
		log.Fatalf("Failed to connect to next node after retries: %v", err)
	}
	defer conn.Close()

	client := pb.NewTokenRingServiceClient(conn)
	token := &pb.Token{
		Holder:    holder,
		Timestamp: int32(time.Now().Unix()),
	}

	_, err = client.PassToken(context.Background(), token)
	if err != nil {
		log.Printf("Failed to pass token to next node: %v", err)
	} else {
		log.Printf("%s passed the token to next node at %s:%s", n.name, n.nextNodeIP, n.nextNodePort)
	}
}

// StartGRPCServer starts the gRPC server for the node
func (n *Node) StartGRPCServer() {
	listener, err := net.Listen("tcp", ":"+n.port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", n.port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTokenRingServiceServer(grpcServer, n)
	log.Printf("%s is listening on port %s", n.name, n.port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

func main() {
	logFile, err := os.OpenFile("../consensus-log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	if len(os.Args) != 5 {
		log.Fatalf("Usage: %s <nodeName> <port> <nextNodeIP> <nextNodePort>", os.Args[0])
	}
	nodeName := os.Args[1]
	port := os.Args[2]
	nextNodeIP := os.Args[3]
	nextNodePort := os.Args[4]

	// Add node to the queue
	AddNodeToQueue(nodeName)

	// Determine if this node should start with the token
	hasToken := ShouldNodeStartWithToken(nodeName)

	// Create a new node
	node := NewNode(nodeName, port, nextNodeIP, nextNodePort, hasToken)

	// Start the gRPC server
	go node.StartGRPCServer()

	// Example token request (emulate critical section access)
	for {
		time.Sleep(10 * time.Second)
		if node.hasToken {
			log.Printf("%s entering critical section", node.name)
			time.Sleep(3 * time.Second) // Simulate critical section operation
			log.Printf("%s leaving critical section", node.name)
			node.hasToken = false
			node.PassTokenToNext(node.name) // Pass the token to the next node
		} else {
			log.Printf("%s does not have the token", node.name)
		}
	}
}

func AddNodeToQueue(nodeName string) {
	nodeQueue = append(nodeQueue, nodeName) // Add the node to the queue
}

// Check if the node should start with the token
func ShouldNodeStartWithToken(nodeName string) bool {
	return len(nodeQueue) == 1 && nodeQueue[0] == nodeName // Only the first node in the queue gets the token
}
