# `APRICOT`

**A** **p**rocessor and **r**epos**i**tory for **co**nfiguration **t**emplates

The `o2-apricot` binary implements a centralized configuration (micro)service for ALICE O².

```
Usage of bin/o2-apricot:
      --backendUri string   URI of the Consul server or YAML configuration file (default "consul://127.0.0.1:8500")
      --listenPort int      Port of apricot server (default 32101)
      --verbose             Verbose logging
```

Protofile: [apricot.proto](protos/apricot.proto)
