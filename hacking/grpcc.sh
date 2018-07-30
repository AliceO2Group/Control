#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )" #parent dir of this script
DIR="$(dirname "$DIR")"                                 #up one level
cd $DIR
CMD="grpcc -i --proto core/protos/octlserver.proto --address 127.0.0.1:47102"
$CMD
