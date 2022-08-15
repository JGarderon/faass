#!/usr/bin/env bash

[ -f faass ] && rm faass ; GOPATH=$(pwd) go build -v -o ./faass || exit 1 

[ ! -f ./conf.json ] && ./faass -prepare 

./faass -testlogger "texte d'essai" -conf ./conf.json 
