#!/usr/bin/env bash

[ ! -d ./tmp ] && mkdir ./tmp 

[ ! -f ./tmp/example-function.py ] && ln ./example-function.py ./tmp/example-function.py 

[ -f faass ] && rm faass ; GOPATH=$(pwd) go build -v -o ./faass || exit 1 

[ ! -f ./conf.json ] && ./faass -prepare 

./faass -testlogger "texte d'essai" -conf ./conf.json 
