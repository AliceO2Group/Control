# octl

## Getting started
Install minimesos, then:
```bash
% cd minimesos
% minimesos up
% cd ..
% make
% bin/octld -url http://172.17.0.5:5050/api/v1/scheduler -executor bin/octl-executor -verbose -server.address `dig +short $HOSTNAME`
```