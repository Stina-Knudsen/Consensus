# Hand-in 4: Consensus

**Welcome to Stanniol's implementation**

Once the repository has been cloned and have opened the program, you should be at the root of this repository.
The instructions assume you start at the root.

1. Step is to set up the nodes
- open a terminal
- cd node
- go run node.go <name> <port> <nextNodeIP> <nextNodePort>
- repeat the steps as many times as you want nodes

Eg. if you want to run the program with three nodes, you need to open three terminals
and run the following commands:
- go run node.go Node1 5001 127.0.0.1 5002
- go run node.go Node2 5002 127.0.0.1 5003
- go run node.go Node3 5003 127.0.0.1 5001
