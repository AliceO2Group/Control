#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #parent dir of this script
DIR="$(dirname "$DIR")"                                 #up one level
cd $DIR

#bin/o2control-core --mesosUrl http://aidrefsrv21.cern.ch:5050/api/v1/scheduler --executor /opt/alisw/el7/Control-Core/v0.9.4-1/bin/o2control-executor --verbose --coreConfigurationUri "file://hacking/config.yaml" --veryVerbose
bin/o2control-core --coreConfigurationUri "file://hacking/settings.yaml"
#bin/o2control-core --coreConfigurationUri file://hacking/settings.yaml.example
#bin/o2control-core --coreConfigurationUri consul://aido2cnf01:8500/test/kostas/o2/aliecs/settingsv4
