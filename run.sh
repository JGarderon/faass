#!/usr/bin/env bash

[ ! -d ./tmp ] && mkdir ./tmp 

[ ! -f ./tmp/example-function.py ] && ln ./example-function.py ./tmp/example-function.py 

buildFlags=""

ConfDirTmp="./tmp"
buildFlags="$buildFlags -X configuration.ConfDirTmp=$ConfDirTmp"

ConfDirContent="./content"
buildFlags="$buildFlags -X configuration.ConfDirContent=$ConfDirContent"

ConfPrefix="lambda"
buildFlags="$buildFlags -X configuration.ConfPrefix=$ConfPrefix"

[ -f faass ] \
	&& rm faass ; \
		GOPATH=$(pwd) go build \
			-ldflags "$buildFlags" \
			-v \
			-o ./faass \
	|| exit 1

[ ! -f ./conf.json ] && ./faass -prepare 

./faass -testlogger "texte d'essai" -conf ./conf.json 
