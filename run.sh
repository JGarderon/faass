#!/usr/bin/env bash

[ -f faass ] && rm faass ; GOPATH=$(pwd) go build -v -o ./faass || exit 1 

[ ! -f ./conf.json ] && ./faass -prepare 

./faass -testlogger 

exit 0 

./faass  --conf ./conf.json # --prepare true

# curl -k https://localhost:9090/redirect && curl -k https://localhost:9090/redirect/ && curl -k https://localhost:9090/redirect/hello

