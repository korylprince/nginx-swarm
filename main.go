package main

import (
	"log"
)

func main() {
	monitor, err := NewMonitor()
	if err != nil {
		log.Fatalln("Error creating monitor:", err)
	}
	monitor.Run()
}
