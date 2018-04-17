# octl with Minimesos

## Getting started
Install minimesos, then:
```bash
% cd minimesos
% minimesos up
% cd ..
% make
% bin/octld -mesos.url http://172.17.0.5:5050/api/v1/scheduler -executor.binary bin/octl-executor -verbose
```