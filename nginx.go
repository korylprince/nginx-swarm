package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

//NGINXConfigPath is the path to the active NGINX configuration
const NGINXConfigPath = "/nginx.conf"

//NGINX represents an NGINX process
type NGINX struct {
	*exec.Cmd
}

//Reload causes NGINX to reload it's configuration
func (n *NGINX) Reload() error {
	return n.Process.Signal(syscall.SIGHUP)
}

//NewConfig writes the given configuration
func (n *NGINX) NewConfig(config []byte) error {
	f, err := os.Create(NGINXConfigPath)
	if err != nil {
		return fmt.Errorf("Error creating %s: %v", NGINXConfigPath, err)
	}

	bytes, err := f.Write(config)
	if err != nil {
		return fmt.Errorf("Error writing %s: %v", NGINXConfigPath, err)
	}

	if Debug {
		log.Println("DEBUG: wrote", bytes, "bytes to", NGINXConfigPath)
	}

	return nil
}

//NewNGINX creates a new NGINX
func NewNGINX() (*NGINX, error) {
	path, err := exec.LookPath("nginx")
	if err != nil {
		return nil, fmt.Errorf("Error finding nginx path: %v", err)
	}

	args := append([]string{"-c", NGINXConfigPath}, os.Args[1:]...)

	c := exec.Command(path, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return &NGINX{c}, nil
}
