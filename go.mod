module github.com/AliceO2Group/Control

go 1.12

// Hack to build go-md2man needed by viper
// Issue: https://github.com/cpuguy83/go-md2man/issues/58
replace github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2

require (
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6
	github.com/armon/go-metrics v0.3.0
	github.com/beorn7/perks v1.0.1
	github.com/briandowns/spinner v1.8.0
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.18+incompatible
	github.com/coreos/go-semver v0.3.0
	github.com/emirpasic/gods v1.12.0
	github.com/fatih/color v1.9.0
	github.com/gdamore/tcell v1.3.0
	github.com/gobwas/glob v0.2.3
	github.com/gogo/protobuf v1.3.1
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/uuid v1.1.1
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/hashicorp/consul/api v1.3.0
	github.com/hashicorp/go-immutable-radix v1.1.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/memberlist v0.1.4 // indirect
	github.com/hashicorp/serf v0.8.5 // indirect
	github.com/hpcloud/tail v1.0.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/json-iterator/go v1.1.9
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/looplab/fsm v0.1.0
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-colorable v0.1.4
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-runewidth v0.0.8
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mesos/mesos-go v0.0.0-20190717023829-56ac038085ac
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/naoina/go-stringutil v0.1.0
	github.com/naoina/toml v0.1.1
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.9.1
	github.com/prometheus/procfs v0.0.8
	github.com/rivo/tview v0.0.0-20200108161608-1316ea7a4b35
	github.com/russross/blackfriday v2.0.0+incompatible
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/src-d/gcfg v1.4.0
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/ugorji/go v1.1.7 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xlab/treeprint v1.0.0
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20200117160349-530e935923ad
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
	golang.org/x/sys v0.0.0-20200121082415-34d275377bf9
	golang.org/x/text v0.3.2
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
	gopkg.in/warnings.v0 v0.1.2
	gopkg.in/yaml.v2 v2.2.7
)
