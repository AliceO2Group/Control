#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #parent dir of this script
DIR="$(dirname "$DIR")"                                 #up one level
cd $DIR

bin/o2control-core --mesosUrl http://m1.dcos:5050/api/v1/scheduler --executor /vagrant/go/src/github.com/AliceO2Group/Control/bin/o2control-executor --verbose --globalConfigurationUri "file://hacking/config.yaml" --veryVerbose
#bin/o2control-core --coreConfigurationUri file://hacking/settings.yaml.example
#bin/o2control-core --coreConfigurationUri consul://aido2cnf01:8500/test/kostas/o2/aliecs/settingsv4