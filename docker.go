package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

//NewClient returns a docker client or an error if one occurred creating it. The client should be closed.
func NewClient() (*client.Client, error) {

	cli, err := client.NewClientWithOpts(client.WithVersion("1.24"))
	if err != nil {
		return nil, fmt.Errorf("Error creating docker client: %v", err)
	}

	return cli, nil
}

//GetServiceConfigs returns the ServiceConfigs for a docker swarm
func GetServiceConfigs() ([]*ServiceConfig, error) {
	cli, err := NewClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	ctx := context.Background()

	svcs, err := cli.ServiceList(ctx, types.ServiceListOptions{})

	if err != nil {
		return nil, fmt.Errorf("Error getting docker service list: %v", err)
	}

	var scs []*ServiceConfig

outer:
	for _, svc := range svcs {

		sc := &ServiceConfig{Name: svc.Spec.Name, ID: svc.ID}

		labels := svc.Spec.Annotations.Labels

		if network, ok := labels["nginx.network"]; ok {
			sc.Network = network
		}

		if port, ok := labels["nginx.port"]; ok {
			for _, portStr := range strings.Split(port, ",") {
				p, parseErr := strconv.Atoi(strings.TrimSpace(portStr))
				if parseErr != nil {
					if Debug {
						log.Printf("DEBUG: couldn't parse \"%s\" service \"nginx.port=%s\": %v; skipping\n", svc.Spec.Name, port, parseErr)
					}

					continue outer
				}

				sc.Ports = append(sc.Ports, p)
			}
		}

		if listenIP, ok := labels["nginx.listenIP"]; ok {
			for _, ipStr := range strings.Split(listenIP, ",") {
				ip := net.ParseIP(strings.TrimSpace(ipStr))
				if ip == nil {
					if Debug {
						log.Printf("DEBUG: couldn't parse \"%s\" service \"nginx.listenIP=%s\": %v; skipping\n", svc.Spec.Name, listenIP, errors.New("address invalid"))
					}
					continue outer
				}

				sc.ListenIPs = append(sc.ListenIPs, ip)
			}

		}

		if port, ok := labels["nginx.listenPort"]; ok {
			for _, portStr := range strings.Split(port, ",") {
				p, parseErr := strconv.Atoi(strings.TrimSpace(portStr))
				if parseErr != nil {
					if Debug {
						log.Printf("DEBUG: couldn't parse \"%s\" service \"nginx.listenPort=%s\": %v; skipping\n", svc.Spec.Name, port, parseErr)
					}
					continue outer
				}

				sc.ListenPorts = append(sc.ListenPorts, p)
			}
		}

		if proto, ok := labels["nginx.listenProto"]; ok {
			for _, protoStr := range strings.Split(proto, ",") {
				protoStr = strings.TrimSpace(strings.ToLower(protoStr))
				if protoStr == "tcp" {
					sc.ListenProtos = append(sc.ListenProtos, ProtocolTCP)
				} else if protoStr == "udp" {
					sc.ListenProtos = append(sc.ListenProtos, ProtocolUDP)
				} else {
					if Debug {
						log.Printf("DEBUG: couldn't parse \"%s\" service \"nginx.listenProto=%s\": %v; skipping\n", svc.Spec.Name, proto, errors.New("protocol must be tcp or udp"))
					}

					continue outer
				}
			}
		}

		if err = sc.Validate(); err != nil {
			if Debug {
				log.Printf("DEBUG: couldn't validate \"%s\" service configuration: %v; skipping\n", svc.Spec.Name, err)
			}
			continue outer
		}

		scs = append(scs, sc)
	}

	return scs, nil
}

//GetServiceAddresses returns the container IP addresses for a service
func GetServiceAddresses(serviceID, network string) ([]net.IP, error) {
	cli, err := NewClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	ctx := context.Background()

	filter := filters.NewArgs()
	filter.Add("service", serviceID)
	filter.Add("desired-state", "running")

	tasks, err := cli.TaskList(ctx, types.TaskListOptions{Filters: filter})
	if err != nil {
		return nil, fmt.Errorf("Error getting docker task list for service id %s: %v", serviceID, err)
	}

	var ips []net.IP

	for _, task := range tasks {
		for _, attach := range task.NetworksAttachments {
			if attach.Network.Spec.Name == network {
				for _, addr := range attach.Addresses {
					ip, _, err := net.ParseCIDR(addr)
					if err != nil {
						return nil, fmt.Errorf("Error parsing address (%s) for service id %s: %v", addr, serviceID, err)
					}

					ips = append(ips, ip)
				}
			}
		}
	}
	return ips, nil
}
