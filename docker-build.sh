#!/bin/bash

version=$1

docker build --no-cache --build-arg "VERSION=$version" --tag "korylprince/nginx-swarm:$version" .

docker push "korylprince/nginx-swarm:$version"
