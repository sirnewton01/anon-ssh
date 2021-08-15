#!/bin/bash
docker run --rm -v `pwd`:/usr/src/app -w /usr/src/app golang:1.16 /usr/src/app/build_app.sh

