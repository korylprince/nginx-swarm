package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

//WatchInterval is the interval Docker Swarm is polled
const WatchInterval = 15 * time.Second

//GetNewConfiguration returns a new NGINX configuration generated from docker, or an error if one occurred
func GetNewConfiguration() (*NGINXConfig, error) {
	scs, err := GetServiceConfigs()
	if err != nil {
		return nil, fmt.Errorf("Error getting docker service configurations: %v", err)
	}

	config := new(NGINXConfig)

	for _, sc := range scs {
		nc, err := sc.NGINXConfig()
		if err != nil {
			return nil, fmt.Errorf("Error converting %s service configuration to nginx configuration: %v", sc.Name, err)
		}

		config.Proxies = append(config.Proxies, nc.Proxies...)
	}

	config.SortProxies()

	return config, nil
}

//Monitor represents a service that manages NGINX and monitors a Docker Swarm
type Monitor struct {
	hash  string
	nginx *NGINX
}

//NewMonitor creates a new Monitor
func NewMonitor() (*Monitor, error) {
	config, err := GetNewConfiguration()
	if err != nil {
		log.Fatalln("ERROR: couldn't generate NGINX configuration:", err)
	}

	j, err := json.Marshal(config)
	if err != nil {
		log.Fatalln("ERROR: couldn't marshal NGINX configuration to json:", err)
	}

	bytes, err := config.Marshal()
	if err != nil {
		log.Fatalln("ERROR: couldn't marshal NGINX configuration:", err)
	}

	log.Println("INFO: new configuration:", string(j))

	nginx, err := NewNGINX()
	if err != nil {
		log.Fatalln("ERROR: couldn't create NGINX process:", err)
	}

	err = nginx.NewConfig(bytes)
	if err != nil {
		log.Fatalln("ERROR: couldn't write NGINX configuration:", err)
	}

	err = nginx.Start()
	if err != nil {
		log.Fatalln("ERROR: couldn't start NGINX process:", err)
	}

	log.Println("INFO: NGINX started with PID", nginx.Process.Pid)

	m := &Monitor{
		hash:  fmt.Sprintf("%x", sha256.Sum256(bytes)),
		nginx: nginx,
	}

	go m.nginxWatchdog()

	return m, nil
}

func (m *Monitor) nginxWatchdog() {
	err := m.nginx.Wait()
	log.Println("ERROR: NGINX process died:", err)
	os.Exit(1)
}

//Run monitors Docker Swarm and makes configuration changes to NGINX
func (m *Monitor) Run() {
	for {
		time.Sleep(WatchInterval)

		if Debug {
			log.Println("DEBUG: polling docker")
		}

		config, err := GetNewConfiguration()
		if err != nil {
			log.Println("WARNING: couldn't generate NGINX configuration:", err)
			continue
		}

		j, err := json.Marshal(config)
		if err != nil {
			log.Fatalln("ERROR: couldn't marshal NGINX configuration to json:", err)
		}

		bytes, err := config.Marshal()
		if err != nil {
			log.Fatalln("ERROR: couldn't marshal NGINX configuration:", err)
		}

		if Debug {
			log.Println("DEBUG: new configuration:", string(j))
		}

		hash := fmt.Sprintf("%x", sha256.Sum256(bytes))

		if hash == m.hash {
			if Debug {
				log.Println("DEBUG: no configuration change; skipping")
			}
			continue
		}

		if !Debug {
			log.Println("INFO: new configuration:", string(j))
		}

		err = m.nginx.NewConfig(bytes)
		if err != nil {
			log.Println("WARNING: couldn't write NGINX configuration:", err)
			continue
		}

		err = m.nginx.Reload()
		if err != nil {
			log.Println("WARNING: couldn't reload NGINX configuration:", err)
		}

		if Debug {
			log.Println("DEBUG: configuration reloaded")
		}

		m.hash = hash
	}
}
