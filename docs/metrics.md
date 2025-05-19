<!-- -->
<!-- === This file is part of ALICE O² === -->
<!-- -->
<!-- Copyright 2025 CERN and copyright holders of ALICE O². -->
<!-- Author: Michal Tichak <michal.tichak@cern.ch> -->
<!-- -->
<!-- This program is free software: you can redistribute it and/or modify -->
<!-- it under the terms of the GNU General Public License as published by -->
<!-- the Free Software Foundation, either version 3 of the License, or -->
<!-- (at your option) any later version. -->
<!-- -->
<!-- This program is distributed in the hope that it will be useful, -->
<!-- but WITHOUT ANY WARRANTY; without even the implied warranty of -->
<!-- MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the -->
<!-- GNU General Public License for more details. -->
<!-- -->
<!-- You should have received a copy of the GNU General Public License -->
<!-- along with this program.  If not, see <http://www.gnu.org/licenses/>. -->
<!-- -->
<!-- In applying this license CERN does not waive the privileges and -->
<!-- immunities granted to it by virtue of its status as an -->
<!-- Intergovernmental Organization or submit itself to any jurisdiction. -->
<!--/ -->

# Metrics in ECS

## Overview and simple usage

ECS core implements metrics in folder common/monitoring. Metrics stack in ALICE
experiment is based on influxdb and Telegraf, where Telegraf scrapes metrics
from given http endpoint in arbitrary format and sends it into the influxdb database
instance. In order to expose endpoint with metrics we can set cli parameter `metricsEndpoint`,
which is in format `[port]/[endpoint]` (default: `8088/ecsmetrics`).
After running core with given parameter we can scrape this endpoint with eg. curl:

```
curl http://127.0.0.1:8088/ecsmetrics
```

Result of this command is for example:

```
kafka,subsystem=ECS,topic=aliecs.run send_bytes=1638400u,sent_messages=42u 1746457955000000000
```

This format is called [influx line protocol](https://docs.influxdata.com/influxdb/cloud/reference/syntax/line-protocol/).
We will use this example to introduce influxdb metrics format. Every line is
one metric and every metric is composed from multiple parts separated by commas
and spaces:

1) name of measurement - (required) string on the beginning of the line (`golangruntimemetrics`)
2) comma
3) tags - (optional) key-value list separated by commas. `subsystem=ECS,topic=aliecs.run` is
the example of key-value list, where subsystem and topic are the keys,
ECS and aliecs.run are values.
4) space - divides measurement and tags from the fields holding measurement data
5) fields - (required) actual values in same format as tags. We support
`int64`, `uint64` and `float64` values (`/sched/goroutines:goroutines=42u`)
6) space - divides fields and timestamp
7) timestamp - (optional) int64 value of unix timestamp in ns

In order to provide support for this format we introduced Metric structure in [common/monitoring/metric.go](https://github.com/AliceO2Group/Control/blob/master/common/monitoring/metric.go).
Following code shows how to create a Metric with `measurement` as measurement name,
one tag `tag1=val1` and field `field1=42u`:

```go
m := monitoring.NewMetric("measurement", time.Now())
m.AddTag("tag1", "val1")
m.SetFieldUInt64("field1", 42)
```

However we also need to be able to store metrics, so these can be scraped correctly.
This mechanism is implemented in [common/monitoring/monitoring.go](https://github.com/AliceO2Group/Control/blob/master/common/monitoring/monitoring.go).
Metrics endpoint is run by calling `Run(port, endpointName)`. As this method is blocking
it is advised to call it from `goroutine`. After this method is called we can than send
metrics via methods `Send` and `SendHistogrammable`. If you want to send simple metrics
(eg. counter of messages sent) you are advised to use simple `Send`. However, if you are
interested into having percentiles reported from metrics you should use
`SendHistogrammable`.

```go
go monigoring.Run(8088, "/metrics")
m := monitoring.NewMetric("measurement", time.Now())
m.AddTag("tag1", "val1")
m.SetFieldUInt64("field1", 42)
monigoring.Send(&m)
monigoring.SendHistogrammable(&m)
monigoring.Stop()
```

Example for this use-case is duration of some function,
eg. measure sending batch of messages. If we want the best coverage of metrics possible
we can combine both of these to measure amount of messages send per batch and
also measurement duration of the send. For example in code you can take a look actual
actual code in [writer.go](https://github.com/AliceO2Group/Control/blob/master/common/event/writer.go) where we are sending multiple
fields per metric and demonstrate full potential of these metrics.

Previous code example will result in following metrics to be reported:

```
measurement,tag1=val1 field1=1 [timestamp]
measurement,tag1=val1 field1_mean=1,field1_median=1,field1_min=1,field1_p10=1,field1_p30=1,field1_p70=1,field1_p90=1,field1_max=1,field1_count=1,field1_poolsize=1 [timestamp]
```

In following text we will talk about aggregating over time interval
which is always 1 second.

First metric is self explanatory, but we can see that Histogrammable metric
reports multiple percentiles, mean, min and max. These values can be the same if
we don't receive send values during 1 second. Moreover it reports `count`
and `poolsize`, where `count` describes number of times this metric was sent
and `poolsize` is internal metric, which will be described later.

## Types and aggregation of metrics

### Metric types

We mentioned in previous part that there are two ways how to send metrics in ECS
resulting in two different outcomes, based on these outcomes we talk about two
metric types:

1) Counter - `Send`
2) Histogrammable - `SendHistogrammable`

Creation of both metrics is done by `NewMetric(measurement, timestamp)`,
so both use same object `Metric`. The distinction is done by Send methods:
`Send` and `SendHistogrammable`.

Simple `Send(*Metric)`is used to just store metric with given tags and fields without
creating any other information. This metric is just aggregated
(more about it later).

Sending metric through `SendHistogrammable(*Metric)`
will result in metric being added into the collection of metrics with same
measurement and tags. This collection is used to compute percentiles
(10, 30, 50, 70, 90), min, max and count (count of elements added into
the collection since last reset). These values are reported as fields
by appending strings to the name of original field:

| Meaning | Appended field  |
| --------|-------------|
| 10-percentile | field_10p |
| 30-percentile | field_30p |
| 70-percentile | field_70p |
| 90-percentile | field_90p |
| median (50-percentile) | field_median |
| mean | field_mean |
| min | field_min |
| max | field_max |
| count | field_count |

### Aggregation

To reduce network bandwidth and RAM usage, we aggregate all metrics.
This is done by grouping them into one-second intervals based on:

- Timestamps rounded down to the nearest second
- Measurement name
- Tags

This aggregation method applies to all metric types.

If multiple metrics share the same measurement name and tags but have timestamps
less than one second apart, they will be grouped into a single bucket, using the
timestamp rounded down to the nearest second.

However, if the measurement name or tags even slightly differ the metrics
will not be grouped and will remain separate. Also, if a metric is missing
a tag present in the others, it won't be aggregated with them.

```
notaggregated,tag1=val1 fields1=1i 1000000123
aggregated,tag1=val1 fields1=1i 1000000001
aggregated,tag1=val1 fields1=1i 1000000021
aggregated,tag1=val1 fields1=1i,fields2=1i 1000000021
aggregated,tag1=val1,tag2=val2 fields1=1i 1000030021
aggregated,tag1=val1 fields1=2i 2000000021
```

If all of these metrics are send and thus aggregated we will result in following:

```
notaggregated,tag1=val1 fields1=1i 1000000000
aggregated,tag1=val1 fields1=3i,fields2=1i 1000000000
aggregated,tag1=val1,tag2=val2 fields1=1i 1000000000
aggregated,tag1=val1 fields1=2i 2000000000
```

Explanation:

- `notaggregated` is unique measurement.
- `aggregated` with one tag will have value of `fields1` equal to 3
as three metrics fell into the same timestamp bucket and `fields2` was aggregated into the same metric
as tags and measurement were the same.
- `aggregated` with either multiple tags or timestamp which is over 2s
cannot be aggregated anywhere and are held as unique values.

The same would happen Histogrammables, except that the aggregation would not be addition
if different points, but creating statistical report as mentioned in previous part.

## Implementation details

### Event loop

In order to send metrics from unlimited amount of goroutines, we need to have
robust and thread-safe mechanism. It is implemented in [common/monitoring/monitoring.go](https://github.com/AliceO2Group/Control/blob/master/common/monitoring/monitoring.go)
as event loop (`eventLoop`) that reads data from two buffered channels (`metricsChannel` and `metricsHistosChannel`)
with one goroutine. Apart from reading messages from these two channels event loop also handles
scraping requests from `http.Server` endpoint. As the http endpoint is called by a
different goroutine than the one processing event loop, we use another two channels:
`metricsRequestedChannel` which is used by the endpoint to request current metrics.
Transformed metrics are sent via `metricsExportedToRequest` back to the endpoint.

Methods `Send` and `SendHistogrammable` write to the corresponding channels,
which are consumed by event loop.

### Hashing to aggregate

In order to correctly implement behaviour described in the part about Aggregation
we use the same implementation in two container aggregating objects
`MetricsAggregate`, `MetricsReservoirSampling` implemented in files
[common/monitoring/metricsaggregate.go](https://github.com/AliceO2Group/Control/blob/master/common/monitoring/metricsaggregate.go)
and [metricsreservoirsampling.go](https://github.com/AliceO2Group/Control/blob/master/common/monitoring/metricsreservoirsampling.go)
in the same directory. The implementation is done as different buckets
in map with distinct keys (`metricsBuckets`). These keys need to be unique
according to the timestamp and tags. We use struct `key` composed
from `time.Time` and `maphash.Hash`. Hash was chosen so we don't have
to compare arbitrary amount of tags, where keys and it's values
must be compared piece by piece. If we are inserting new bucket into `metricsBuckets`
we create new key by rounding down timestamp `time.Unix(metric.timestamp.Unix(), 0)`
and hashing all tags and their values. This will result to unique buckets
distinguished by timestamp and tags collection.

However there is a potential problem:
We are storing tags as unsorted slice in `Metric`, so it is possible to create
two metrics with same tags, but in different order. These will result
in different hashes as maphash is order-dependent.

### Sampling reservoir

We are computing percentiles from all metrics sent via `SendHistogrammable`
method, but we are computing these from streaming data with unknown
limits, so we cannot easily create histogram. However there exist simple
solution called sampling reservoir and is discussed in [this wiki article](https://en.wikipedia.org/wiki/Reservoir_sampling).
It uses easy principle where every value from streaming data must have
the same probability of staying inside fixed buffer called reservoir.
