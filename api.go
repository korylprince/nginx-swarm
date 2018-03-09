package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sort"
)

//Protocol represents an IP Protocol
type Protocol int

const (
	// ProtocolTCP TCP
	ProtocolTCP Protocol = iota
	// ProtocolUDP UDP
	ProtocolUDP
)

func (p Protocol) String() string {
	if p == ProtocolTCP {
		return "tcp"
	}

	return "udp"
}

//ServiceConfig represents a docker swarm service configuration
type ServiceConfig struct {
	Name         string
	ID           string
	Network      string     //nginx.network
	Ports        []int      //nginx.port
	ListenIPs    []net.IP   //nginx.listenIP
	ListenPorts  []int      //nginx.listenPort
	ListenProtos []Protocol //nginx.listenProto
}

//Validate validates the ServiceConfig and returns an error if it's not valid
func (c *ServiceConfig) Validate() error {
	if c.Name == "" {
		return errors.New("Name must be set")
	}

	if c.ID == "" {
		return errors.New("ID must be set")
	}

	if c.Network == "" {
		return errors.New("nginx.network must be set")
	}

	//calculate max parameter length
	max := len(c.Ports)

	if l := len(c.ListenIPs); l > max {
		max = l
	}
	if l := len(c.ListenPorts); l > max {
		max = l
	}
	if l := len(c.ListenProtos); l > max {
		max = l
	}

	//verify parameter lengths
	if l := len(c.Ports); l == 1 {
		for i := 1; i < max; i++ {
			c.Ports = append(c.Ports, c.Ports[0])
		}
	} else if l < max {
		return fmt.Errorf("nginx.port length mismatch. Expected %d or 1, got %d", max, l)
	}

	if l := len(c.ListenIPs); l == 1 {
		for i := 1; i < max; i++ {
			c.ListenIPs = append(c.ListenIPs, c.ListenIPs[0])
		}
	} else if l < max {
		return fmt.Errorf("nginx.listenIP length mismatch. Expected %d or 1, got %d", max, l)
	}

	if l := len(c.ListenPorts); l == 1 {
		for i := 1; i < max; i++ {
			c.ListenPorts = append(c.ListenPorts, c.ListenPorts[0])
		}
	} else if l < max {
		return fmt.Errorf("nginx.listenPort length mismatch. Expected %d or 1, got %d", max, l)
	}

	if l := len(c.ListenProtos); l == 1 {
		for i := 1; i < max; i++ {
			c.ListenProtos = append(c.ListenProtos, c.ListenProtos[0])
		}
	} else if l < max {
		return fmt.Errorf("nginx.listenProto length mismatch. Expected %d or 1, got %d", max, l)
	}

	return nil
}

//NGINXConfig returns an *NGINXConfig converted from c
func (c *ServiceConfig) NGINXConfig() (*NGINXConfig, error) {
	addrs, err := GetServiceAddresses(c.ID, c.Network)
	if err != nil {
		return nil, fmt.Errorf("Error getting service (%s) addresses: %v", c.Name, err)
	}

	ng := new(NGINXConfig)

	for i := 0; i < len(c.Ports); i++ {
		p := &Proxy{
			Name:        c.Name,
			ListenIP:    c.ListenIPs[i],
			ListenPort:  c.ListenPorts[i],
			ListenProto: c.ListenProtos[i],
		}

		for _, addr := range addrs {
			p.Servers = append(p.Servers, &Server{
				Address: addr,
				Port:    c.Ports[i],
			})
		}

		p.SortServers()

		ng.Proxies = append(ng.Proxies, p)

	}

	return ng, nil
}

//Server represents a proxy backend (container)
type Server struct {
	Address net.IP
	Port    int
}

//SortKey provides a sortable key for s
func (s *Server) SortKey() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

//Proxy represents proxied service
type Proxy struct {
	Name        string
	ListenIP    net.IP
	ListenPort  int
	ListenProto Protocol
	Servers     []*Server
}

//SortServers sorts the server list in p
func (p *Proxy) SortServers() {
	sort.Slice(p.Servers, func(i, j int) bool {
		return p.Servers[i].SortKey() < p.Servers[j].SortKey()
	})
}

//SortKey provides a sortable key for p
func (p *Proxy) SortKey() string {
	return fmt.Sprintf("%s %s:%d/%s", p.Name, p.ListenIP, p.ListenPort, p.ListenProto)
}

//IsUDP is a helper function for templating
func (p *Proxy) IsUDP() bool {
	return p.ListenProto == ProtocolUDP
}

//NGINXConfig represents an nginx configuration
type NGINXConfig struct {
	Proxies []*Proxy
}

//SortProxies sorts the proxy list in c
func (c *NGINXConfig) SortProxies() {
	sort.Slice(c.Proxies, func(i, j int) bool {
		return c.Proxies[i].SortKey() < c.Proxies[j].SortKey()
	})
}

//Marshal returns the nginx configuration represented by c
func (c *NGINXConfig) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := nginxTmpl.Execute(buf, c)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
