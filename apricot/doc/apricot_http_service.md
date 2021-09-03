## apricot HTTP service

Web server component that implements the Trivial Configuration Endpoint (Apricot TCE). 

It serves JSON and/or plain text structures in order to make essential cluster information available to scripts and other consumers for which the gRPC interface is impractical.

It is strictly **read-only** and only ever responds to `GET`.

### Configuration

To use this feature, the `o2-apricot` running machine has to communicate with Consul via the `--backendUri` option (see [here](apricot.md)).

### Usage and options

To retrieve the information needed regarding FLPs, use the following urls in a web browser.

To retrieve as plain text:
* `your_machine/inventory/flps` or `your_machine/inventory/flps/text`
* `/inventory/detectors/your_detector/flps` or `/inventory/detectors/your_detector/flps/text`

To retrieve as JSON:
* `your_machine/inventory/flps/json`
* `/inventory/detectors/your_detector/flps/json`
