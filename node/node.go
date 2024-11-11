package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	proto "Consensus/grpc"

	"google.golang.org/grpc"
)

type Node struct {
	proto.UnimplementedTokenRingServiceServer
	name         string
	port         string
	nextNodeIP   string
	nextNodePort string
	hasToken     bool
}

var nodeQueue []string

func main() {
	// ready for a log
	logFile, err := os.OpenFile("../consensus-log.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// actual main starts
	if len(os.Args) != 5 {
		log.Fatalf("Usage: %s <nodeName> <port> <nextNodeIP> <nextNodePort>", os.Args[0])
	}
	nodeName := os.Args[1]
	port := os.Args[2]
	nextNodeIP := os.Args[3]
	nextNodePort := os.Args[4]

	AddNodeToQueue(nodeName)

	hasToken := ShouldNodeStartWithToken(nodeName)

	node := NewNode(nodeName, port, nextNodeIP, nextNodePort, hasToken)

	go node.StartGRPCServer()

	for {
		time.Sleep(10 * time.Second)
		if node.hasToken {
			log.Printf("%s entering critical section", node.name)
			time.Sleep(3 * time.Second)
			log.Printf("%s leaving critical section", node.name)
			node.hasToken = false
			node.PassTokenToNext(node.name)
		} else {
			log.Printf("%s does not have the token", node.name)
		}
	}
}

func NewNode(name, port, nextNodeIP, nextNodePort string, hasToken bool) *Node {
	return &Node{
		name:         name,
		port:         port,
		nextNodeIP:   nextNodeIP,
		nextNodePort: nextNodePort,
		hasToken:     true,
	}
}

func (n *Node) RequestToken(ctx context.Context, req *proto.TokenRequest) (*proto.TokenResponse, error) {
	if n.hasToken {
		log.Printf("%s granting token to %s", n.name, req.GetNodeName())
		n.hasToken = false
		n.PassTokenToNext(req.GetNodeName())
		return &proto.TokenResponse{
			Status:  "granted",
			Message: fmt.Sprintf("Token passed to %s", req.GetNodeName()),
		}, nil
	}
	log.Printf("%s does not have the token to grant", n.name)
	return &proto.TokenResponse{
		Status:  "denied",
		Message: "Token not available",
	}, nil
}

func (n *Node) PassToken(ctx context.Context, token *proto.Token) (*proto.TokenResponse, error) {
	log.Printf("%s received the token from %s", n.name, token.GetHolder())
	n.hasToken = true
	return &proto.TokenResponse{
		Status:  "received",
		Message: "Token received",
	}, nil
}

func (n *Node) PassTokenToNext(holder string) {
	var conn *grpc.ClientConn
	var err error
	for i := 0; i < 5; i++ { // Retry up to 5 times
		conn, err = grpc.Dial(fmt.Sprintf("%s:%s", n.nextNodeIP, n.nextNodePort), grpc.WithInsecure())
		if err == nil {
			break
		}
		log.Printf("Failed to connect to next node, retrying... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Failed to connect to next node after retries: %v", err)
	}
	defer conn.Close()

	client := proto.NewTokenRingServiceClient(conn)
	token := &proto.Token{
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

func (n *Node) StartGRPCServer() {
	listener, err := net.Listen("tcp", ":"+n.port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", n.port, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterTokenRingServiceServer(grpcServer, n)
	log.Printf("%s is listening on port %s", n.name, n.port)

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve gRPC server: %v", err)
	}
}

func AddNodeToQueue(nodeName string) {
	nodeQueue = append(nodeQueue, nodeName)
}

func ShouldNodeStartWithToken(nodeName string) bool {
	return len(nodeQueue) == 1 && nodeQueue[0] == nodeName
}
