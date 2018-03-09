package main

import "text/template"

const nginxTmplText = `
daemon off;
error_log /dev/stdout info;

events {
    worker_connections  1024;
}

stream {
    log_format proxy '$time_iso8601  [info] '
        'client: $remote_addr, server: $server_addr:$server_port, '
        'upstream: $upstream_addr/$protocol, '
        'bytes from/to client:$bytes_received/$bytes_sent, '
        'bytes from/to upstream:$upstream_bytes_received/$upstream_bytes_sent';

    access_log /dev/stdout proxy;
	{{ range $proxy := .Proxies}}
	upstream {{$proxy.Name}}_{{$proxy.ListenProto}}_{{$proxy.ListenPort}}_backend {
		hash	$remote_addr;
	{{- range $server := $proxy.Servers}}
		server	{{$server.Address}}:{{$server.Port}};
	{{- end}}
	}
	server {
		listen		{{$proxy.ListenIP}}:{{$proxy.ListenPort}} {{- if $proxy.IsUDP}} udp{{end}};
		proxy_pass	{{$proxy.Name}}_{{$proxy.ListenProto}}_{{$proxy.ListenPort}}_backend;
	}
	{{end}}
}`

var nginxTmpl = template.Must(template.New("nginx").Parse(nginxTmplText))
