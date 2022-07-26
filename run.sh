#!/usr/bin/env bash

[ -f faass ] && rm faass ; go build && ./faass --conf ./conf.json

# curl -k https://localhost:9090/redirect && curl -k https://localhost:9090/redirect/ && curl -k https://localhost:9090/redirect/hello

