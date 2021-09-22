## apricot HTTP service

Web server component that implements the Trivial Configuration Endpoint (Apricot TCE). 

It serves JSON and/or plain text structures in order to make essential cluster information available to scripts and other consumers for which the gRPC interface is impractical.

It is strictly **read-only** and only ever responds to `GET`.

### Configuration

To use this feature, the `o2-apricot` running machine has to communicate with Consul via the `--backendUri` option (see [here](apricot.md)).

### Usage and options

To retrieve the information needed regarding FLPs, use the following urls in a web browser or with `curl` or `wget`.

To retrieve as plain text:
* `http://<apricot-server>/inventory/flps` or `http://<apricot-server>/inventory/flps/text`
* `http://<apricot-server>/inventory/detectors/<detector>/flps` or `http://<apricot-server>/inventory/detectors/<detector>/flps/text`

To retrieve as JSON:
* `http://<apricot-server>/inventory/flps/json`
* `http://<apricot-server>/inventory/detectors/<detector>/flps/json`

### Examples

* With `curl`: `curl http://localhost:32188/inventory/flps`
* With `wget`: `wget http://localhost:32188/inventory/detectors/TST/flps/json -O ~/downloads/test`