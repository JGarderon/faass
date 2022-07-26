#!/usr/bin/env bash

[ -f faass ] && rm faass ; go build && ./faass --prepare true # --conf ./original-conf.json

# curl -k https://localhost:9090/redirect && curl -k https://localhost:9090/redirect/ && curl -k https://localhost:9090/redirect/hello

