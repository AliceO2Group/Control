module github.com/AliceO2Group/Control

go 1.14

// github.com/coreos/bbolt@v1.3.4: parsing go.mod:
//         module declares its path as: go.etcd.io/bbolt
//                 but was required as: github.com/coreos/bbolt
replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.3

// Issue: https://github.com/etcd-io/etcd/issues/11563
replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	google.golang.org/grpc => google.golang.org/grpc v1.30.0
)

// Issue: https://github.com/rivo/tview/issues/416
// tview should be version 0ba8301b415c otherwise peanut will deadlock

require (
	github.com/AlecAivazis/survey/v2 v2.1.1
	github.com/antonmedv/expr v1.4.5
	github.com/armon/go-metrics v0.3.3 // indirect
	github.com/briandowns/spinner v1.11.1
	github.com/coreos/bbolt v1.3.4 // indirect
	github.com/coreos/etcd v3.3.19+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/gdamore/tcell v1.3.0
	github.com/gliderlabs/ssh v0.3.0 // indirect
	github.com/go-git/go-git/v5 v5.1.0
	github.com/go-yaml/yaml v2.1.0+incompatible // indirect
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/uuid v1.1.1
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.3 // indirect
	github.com/hashicorp/consul/api v1.4.0
	github.com/hashicorp/go-hclog v0.14.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.2.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.9.2 // indirect
	github.com/imdario/mergo v0.3.9
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/looplab/fsm v0.1.0
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-colorable v0.1.6 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mesos/mesos-go v0.0.11
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/miekg/dns v1.1.29 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.1 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/naoina/toml v0.1.1
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.12.2
	github.com/onsi/gomega v1.10.1
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7 // indirect
	github.com/prometheus/client_golang v1.5.1
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/rivo/tview v0.0.0-20200219135020-0ba8301b415c
	github.com/rs/xid v1.2.1
	github.com/sanity-io/litter v1.2.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.5.1 // indirect
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc // indirect
	github.com/valyala/fasttemplate v1.1.0
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xlab/treeprint v1.0.0
	go.etcd.io/bbolt v1.3.4 // indirect
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	golang.org/x/sys v0.0.0-20200803210538-64077c9b5642
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/grpc v1.31.0
	google.golang.org/grpc/examples v0.0.0-20200826230536-d31b6710005d // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200506231410-2ff61e1afc86
	sigs.k8s.io/yaml v1.2.0 // indirect
)
