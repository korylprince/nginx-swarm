package main

import (
	"log"
	"os"
	"strings"
)

//Debug is a global debug setting
var Debug = false

func init() {
	for _, arg := range os.Environ() {
		if strings.ToLower(arg) == "debug=true" {
			Debug = true
			log.Println("DEBUG: debugging on")
		}
	}
}
