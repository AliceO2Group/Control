module github.com/AliceO2Group/Control

go 1.14

// github.com/coreos/bbolt@v1.3.4: parsing go.mod:
//         module declares its path as: go.etcd.io/bbolt
//                 but was required as: github.com/coreos/bbolt
replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.3

// Issue: https://github.com/etcd-io/etcd/issues/11563
replace (
	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
	google.golang.org/grpc => google.golang.org/grpc v1.34.0
)

// Issue: https://github.com/rivo/tview/issues/416
// tview should be version 0ba8301b415c otherwise peanut will deadlock

require (
	cloud.google.com/go v0.76.0 // indirect
	cloud.google.com/go/firestore v1.4.0 // indirect
	github.com/AlecAivazis/survey/v2 v2.2.7
	github.com/AliceO2Group/Bookkeeping v0.16.8 // indirect
	github.com/AliceO2Group/Bookkeeping/go-api-client v0.0.0-20210308150404-e78be0de914f
	github.com/BurntSushi/xgb v0.0.0-20210121224620-deaf085860bc // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.4.16 // indirect
	github.com/alecthomas/units v0.0.0-20210208195552-ff826a37aa15 // indirect
	github.com/antlr/antlr4 v0.0.0-20191011202612-ad2bd05285ca // indirect
	github.com/antonmedv/expr v1.8.9
	github.com/aokoli/goutils v1.1.1 // indirect
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6 // indirect
	github.com/armon/go-metrics v0.3.6 // indirect
	github.com/bketelsen/crypt v0.0.3 // indirect
	github.com/briandowns/spinner v1.12.0
	github.com/census-instrumentation/opencensus-proto v0.3.0 // indirect
	github.com/cncf/udpa/go v0.0.0-20210210032658-bff43e8824d0 // indirect
	github.com/coreos/bbolt v1.3.4 // indirect
	github.com/coreos/etcd v3.3.25+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
	github.com/deckarep/golang-set v1.7.1 // indirect
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dmarkham/enumer v1.5.2
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/envoyproxy/go-control-plane v0.9.8 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.4.1 // indirect
	github.com/fatih/color v1.10.0
	github.com/flosch/pongo2/v4 v4.0.2
	github.com/gdamore/tcell/v2 v2.1.0
	github.com/gliderlabs/ssh v0.3.0 // indirect
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-gl/glfw/v3.3/glfw v0.0.0-20201108214237-06ea97f0c265 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible // indirect
	github.com/gobwas/glob v0.2.3
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.2.0 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/gorilla/mux v1.7.3
	github.com/grpc-ecosystem/go-grpc-middleware v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.3 // indirect
	github.com/hashicorp/consul/api v1.8.1
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v0.15.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/imdario/mergo v0.3.11
	github.com/jinzhu/copier v0.2.4
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/kevinburke/ssh_config v0.0.0-20201106050909-4977a11b4351 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/looplab/fsm v0.2.0
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.4 // indirect
	github.com/mesos/mesos-go v0.0.11
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/miekg/dns v1.1.29 // indirect
	github.com/mitchellh/copystructure v1.1.1 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/naoina/toml v0.1.1
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/osamingo/indigo v1.1.0
	github.com/osteele/liquid v1.2.4 // indirect
	github.com/osteele/tuesday v1.0.3 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pborman/uuid v1.2.1
	github.com/pelletier/go-toml v1.8.1 // indirect
	github.com/pkg/sftp v1.12.0 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7 // indirect
	github.com/pressly/chi v0.0.0-00010101000000-000000000000 // indirect
	github.com/prometheus/client_golang v1.9.0
	github.com/pseudomuto/protoc-gen-doc v1.4.1
	github.com/rivo/tview v0.0.0-20210125085121-dbc1f32bb1d0
	github.com/rs/xid v1.2.1
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/afero v1.5.1 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/tmc/grpc-websocket-proxy v0.0.0-20200122045848-3419fae592fc // indirect
	github.com/ugorji/go v1.1.4 // indirect
	github.com/valyala/fasttemplate v1.2.1
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xlab/treeprint v1.0.0
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77 // indirect
	github.com/yuin/goldmark v1.3.2 // indirect
	go.etcd.io/bbolt v1.3.4 // indirect
	go.opencensus.io v0.22.6 // indirect
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad // indirect
	golang.org/x/exp v0.0.0-20210201131500-d352d2db2ceb // indirect
	golang.org/x/image v0.0.0-20201208152932-35266b937fa6 // indirect
	golang.org/x/mobile v0.0.0-20210208171126-f462b3930c8f // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210423082822-04245dca01da
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	google.golang.org/api v0.39.0 // indirect
	google.golang.org/genproto v0.0.0-20210207032614-bba0dbe2a9ea // indirect
	google.golang.org/grpc v1.35.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/grpc/examples v0.0.0-20210210183804-ad24ab52b162 // indirect
	google.golang.org/protobuf v1.25.0
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/ini.v1 v1.62.0 // indirect
	gopkg.in/osteele/liquid.v1 v1.2.4 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/codahale/hdrhistogram => github.com/HdrHistogram/hdrhistogram-go v1.0.1

replace github.com/pressly/chi => github.com/go-chi/chi v1.5.2
