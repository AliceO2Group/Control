## apricot HTTP service

Web server component that implements the REST Configuration Endpoint (Apricot RCE). 

It serves JSON and/or plain text structures in order to make essential cluster information and component configuration available to scripts and other consumers for which the gRPC interface is impractical.

It is strictly **read-only** and only ever responds to `GET` (except for the special purpose `POST` call `/components/_invalidate_cache`, called only by Consul).

### Configuration

To use this feature, the `o2-apricot` running machine has to communicate with Consul via the `--backendUri` option (see [here](apricot.md)).

### Usage and options

The default port is `32188`.

To retrieve the information needed regarding FLPs, use the following urls in a web browser or with `curl` or `wget`.

To retrieve as plain text:
* `http://<apricot-server>/inventory/flps` or `http://<apricot-server>/inventory/flps/text`
* `http://<apricot-server>/inventory/detectors/<detector>/flps` or `http://<apricot-server>/inventory/detectors/<detector>/flps/text`

To retrieve as JSON:
* `http://<apricot-server>/inventory/flps/json`
* `http://<apricot-server>/inventory/detectors/<detector>/flps/json`

To retrieve a component configuration payload, use the following urls in a web browser or with `curl` or `wget`.

* `http://<apricot-server>/components/<component>/<runtype>/<rolename>/<entry>?process=true` - with template processing
* `http://<apricot-server>/components/<component>/<runtype>/<rolename>/<entry>?process=false` - without template processing, returns the entry verbatim

The full API documentation is available at `http://<apricot-server>/docs/` wherever your Apricot instance is running. This documentation interface also allows to perform API calls directly from the browser.

### Examples

* With `curl`: `curl http://localhost:32188/inventory/flps`
* With `wget`: `wget http://localhost:32188/inventory/detectors/TST/flps/json -O ~/downloads/test`