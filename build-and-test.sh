#!/usr/bin/env bash

clear ; clear 

printf "\n---\t\t---\t\t---\t start\n\n"

[ ! -d ./tmp ] && ( mkdir ./tmp && echo "tmp path created" ) || echo "tmp path existent"
[ ! -d ./cache ] && ( mkdir ./cache && echo "cache path created" ) || echo "cache path existent"

[ ! -f ./tmp/example-function.py ] && ( ln ./example-function.py ./tmp/example-function.py && echo "link for example function created" ) || echo "example function existent" 

buildFlags=""

ConfDirTmp="./tmp"
buildFlags="$buildFlags -X configuration.ConfDirTmp=$ConfDirTmp"

ConfDirContent="./content"
buildFlags="$buildFlags -X configuration.ConfDirContent=$ConfDirContent"

ConfPrefix="lambda"
buildFlags="$buildFlags -X configuration.ConfPrefix=$ConfPrefix"

printf "\n---\t\t---\t\t---\t build\n\n"

exec="./tests/functional-testing.py \
	--build \
	--origin-path `pwd` \
	--cache-path `pwd`/cache \
	${@:1}"

[ -f faass ] \
	&&  rm faass ; \
		`$exec` \
	|| exit 1

[ ! -f ./conf.json ] && ./faass -prepare 

printf "\n---\t\t---\t\t---\t functional tests\n\n"

`PYTHONWARNINGS="ignore:Unverified HTTPS request" ./tests/functional-testing.py \
	--run \
	${@:1}`

printf "\n---\t\t---\t\t---\t stop\n\n"
