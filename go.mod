module github.com/AliceO2Group/Control

go 1.18

// github.com/coreos/bbolt@v1.3.4: parsing go.mod:
//         module declares its path as: go.etcd.io/bbolt
//                 but was required as: github.com/coreos/bbolt
replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.6

// Issue: https://github.com/etcd-io/etcd/issues/11563
//replace (
//	github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
//	google.golang.org/grpc => google.golang.org/grpc v1.43.0
//)

// Issue: https://github.com/rivo/tview/issues/416
// tview should be version 0ba8301b415c otherwise peanut will deadlock

require (
	github.com/AlecAivazis/survey/v2 v2.3.5
	github.com/AliceO2Group/Bookkeeping/go-api-client v0.0.0-20220921090341-85645ac18c81
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/antonmedv/expr v1.9.0
	github.com/briandowns/spinner v1.19.0
	github.com/denisbrodbeck/machineid v1.0.1
	github.com/dmarkham/enumer v1.5.5
	github.com/fatih/color v1.13.0
	github.com/flosch/pongo2/v4 v4.0.2
	github.com/gdamore/tcell/v2 v2.5.2
	github.com/go-git/go-git/v5 v5.4.2
	github.com/gobwas/glob v0.2.3
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/hashicorp/consul/api v1.13.1
	github.com/imdario/mergo v0.3.13
	github.com/jinzhu/copier v0.3.5
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/looplab/fsm v0.3.0
	github.com/mesos/mesos-go v0.0.11
	github.com/mitchellh/go-homedir v1.1.0
	github.com/naoina/toml v0.1.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/osamingo/indigo v1.1.0
	github.com/pborman/uuid v1.2.1
	github.com/prometheus/client_golang v1.12.2
	github.com/pseudomuto/protoc-gen-doc v1.5.1
	github.com/rivo/tview v0.0.0-20220731115447-9d32d269593e
	github.com/rs/xid v1.4.0
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/segmentio/kafka-go v0.4.33
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/valyala/fasttemplate v1.2.1
	github.com/xeipuuv/gojsonschema v1.2.0
	github.com/xlab/treeprint v1.1.0
	golang.org/x/crypto v0.0.0-20220722155217-630584e8d5aa
	golang.org/x/net v0.7.0
	golang.org/x/sys v0.5.0
	google.golang.org/grpc v1.50.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.28.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/AliceO2Group/Bookkeeping v0.17.13-0.20220921090341-85645ac18c81 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20220730123233-d6ffb7692adf // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/armon/go-metrics v0.4.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cloudflare/circl v1.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.7 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gliderlabs/ssh v0.3.0 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.2.2 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/serf v0.9.8 // indirect
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/mwitkow/go-proto-validators v0.3.2 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/osamingo/base58 v1.0.0 // indirect
	github.com/pascaldekloe/name v1.0.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.2 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/pseudomuto/protokit v0.2.1 // indirect
	github.com/rivo/uniseg v0.3.1 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/sony/sonyflake v1.0.1-0.20200827011719-848d664ceea4 // indirect
	github.com/spf13/afero v1.9.2 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.4.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xanzy/ssh-agent v0.3.1 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/oauth2 v0.0.0-20220822191816-0ebed06d0094 // indirect
	golang.org/x/term v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	golang.org/x/tools v0.1.12 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20221027153422-115e99e71e1c // indirect
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

replace github.com/codahale/hdrhistogram => github.com/HdrHistogram/hdrhistogram-go v1.0.1

replace github.com/pressly/chi => github.com/go-chi/chi v1.5.2
