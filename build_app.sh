#!/bin/bash

# build all three versions windows, linux, mac, in parallel
go get ./...
make build -j3
