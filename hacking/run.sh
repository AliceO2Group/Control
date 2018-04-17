#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #parent dir of this script
DIR="$(dirname "$DIR")"                                 #up one level
cd $DIR

bin/octld -mesos.url http://m1.dcos:5050/api/v1/scheduler -executor.binary ./bin/octl-executor -verbose -config "file://hacking/example-config.yaml"
