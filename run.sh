#!/usr/bin/env bash

[ -f faass ] && rm faass ; go build && ./faass  --conf ./original-conf.json # --prepare true

# curl -k https://localhost:9090/redirect && curl -k https://localhost:9090/redirect/ && curl -k https://localhost:9090/redirect/hello

