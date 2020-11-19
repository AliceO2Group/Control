#!/bin/zsh

# === This file is part of ALICE O² ===
#
# Copyright 2020 CERN and copyright holders of ALICE O².
# Author: Teo Mrnjavac <teo.m@cern.ch>
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
# In applying this license CERN does not waive the privileges and
# immunities granted to it by virtue of its status as an
# Intergovernmental Organization or submit itself to any jurisdiction.


# Usage example for 200 iterations:
# ./test.sh 200
# The test may take up to 20s per iteration.


# Configuration
HOST="centosvmtest2"
ENVCREATE_FILE="envcreate.csv"
ENVSTART_FILE="envstart.csv"

STARTTIME=$(date +%s.%N) # reset script timer
TIMEFMT=$'total time: %*E' # make zsh function time only output the total time

# CSV append function (really only appends a line for now)
function csv_append() {
    echo "$2" >> $1
}

# If the number of iterations is not passed as argument, we default to 1
if [ -z ${1+x} ]; then ITERATIONS=1; else ITERATIONS=$1; fi

# Prepare create command...
read -r -d '' CMD_CREATE << EOM
coconut e c -w readout-dataflow@master -e '{"hosts":["$HOST"],"dd_enabled":"true","qcdd_enabled":"true"}'
EOM

# Zero the output files
truncate -s 0 $ENVCREATE_FILE
truncate -s 0 $ENVSTART_FILE


# Main loop starts here
echo "iter\tenv      \twarm\tcreate\tstart\tdestroy\troster\tzombies\tshmmon\tshmfiles"
for ((iter=0; iter<$ITERATIONS; iter++)); do


### TEST CODE ### UNCOMMENT FROM HERE ###
# read -r -d '' OUTPUT << EOM
# new environment created with 8 tasks
# environment id:     2MFXZxnvRCj
# state:              CONFIGURED
# root role:          readout-dataflow
# EOM
# read -r -d '' CMD_CREATE << EOM
# echo "$OUTPUT"
# EOM
### END TEST CODE

    OUTPUT=$({time (eval $CMD_CREATE)} 2>&1)

    # sleep 1
    ENV=`echo $OUTPUT|grep "environment id"|awk '{ print $3}'`
    CREATE_TIME=`echo $OUTPUT|grep "total time:"|awk '{ print $3}'`

    # Append measured time to file
    csv_append $ENVCREATE_FILE $CREATE_TIME

    CMD_START="coconut e t -e start $ENV"

### TEST CODE ### UNCOMMENT FROM HERE ###
# read -r -d '' OUTPUT << EOM
# transition complete
# environment id:     2MFXZxnvRCj
# state:              RUNNING
# run number:         154
# EOM
# read -r -d '' CMD_START << EOM
# echo "$OUTPUT"
# EOM
### END TEST CODE

    WARM_START=`ssh root@$HOST pgrep -f 'o2control-executor' -c`

    OUTPUT=$({time (eval $CMD_START)} 2>&1)
    # sleep 1
    START_TIME=`echo $OUTPUT|grep "total time:"|awk '{ print $3}'`

    # Append measured time to file
    csv_append $ENVSTART_FILE $START_TIME

    CMD_DESTROY="coconut e d -f $ENV"
    OUTPUT=$({time (eval $CMD_DESTROY)} 2>&1)
    DESTROY_TIME=`echo $OUTPUT|grep "total time:"|awk '{ print $3}'`

    COCONUT_TASKS_LEFT_OVER="$(( `coconut t l|wc -l` - 1))"
    ZOMBIE_TASKS=`ssh root@$HOST pgrep -f 'OCCPlugin' -c`
    
    # Experimentally determined that shmmonitor needs about 3s to clean up and die
    sleep 3

    SHMMONITOR_RUNNING=`ssh root@$HOST pgrep -f 'fairmq-shmmonitor' -c`

    SHM_LINES=`ssh root@$HOST ls /dev/shm|wc -l`

    # sleep 1
    ssh root@$HOST rm -rf "/dev/shm/*" 

    echo "${(l:3::0:)iter}\t$ENV\t$WARM_START\t$CREATE_TIME\t$START_TIME\t$DESTROY_TIME\t$COCONUT_TASKS_LEFT_OVER\t$ZOMBIE_TASKS\t$SHMMONITOR_RUNNING\t$SHM_LINES"
done

ENDTIME=$(date +%s.%N)
dt=$(echo "$ENDTIME - $STARTTIME" | bc)
dd=$(echo "$dt/86400" | bc)
dt2=$(echo "$dt-86400*$dd" | bc)
dh=$(echo "$dt2/3600" | bc)
dt3=$(echo "$dt2-3600*$dh" | bc)
dm=$(echo "$dt3/60" | bc)
ds=$(echo "$dt3-60*$dm" | bc)

LC_NUMERIC=C printf "${(l:3::0:)ITERATIONS} iterations done in %d:%02d:%02d:%02.4f\n" $dd $dh $dm $ds
