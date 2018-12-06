#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #parent dir of this script
DIR="$(dirname "$DIR")"                                 #up one level
cd $DIR

bin/o2control-core -mesos.url http://m1.dcos:5050/api/v1/scheduler -executor.binary /vagrant/go/src/github.com/AliceO2Group/Control/bin/o2control-executor -verbose -config "file://hacking/config.yaml" -veryVerbose
