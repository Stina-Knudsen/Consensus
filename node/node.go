package main

const (
	held     string = "held"
	released string = "released"
	wanted   string = "wanted"
)

type Node struct {
	state string
}
