module github.com/AliceO2Group/Control

go 1.12

// Hack to build go-md2man needed by viper
// Issue: https://github.com/cpuguy83/go-md2man/issues/58
replace github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2

// Hack to get a working binary needed by viper
// Issue: https://github.com/spf13/viper/issues/644
replace github.com/coreos/etcd => github.com/coreos/etcd v3.3.18+incompatible

require (
	github.com/armon/go-metrics v0.3.0 // indirect
	github.com/briandowns/spinner v1.8.0
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
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
	github.com/imdario/mergo v0.3.8
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/looplab/fsm v0.1.0
	github.com/lucasb-eyer/go-colorful v1.0.3 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-runewidth v0.0.8 // indirect
	github.com/mesos/mesos-go v0.0.0-20190717023829-56ac038085ac
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/naoina/toml v0.1.1
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	github.com/pborman/uuid v1.2.0
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7 // indirect
	github.com/prometheus/client_golang v1.3.0
	github.com/prometheus/common v0.9.1 // indirect
	github.com/rivo/tview v0.0.0-20200108161608-1316ea7a4b35
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xlab/treeprint v1.0.0
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20200117160349-530e935923ad // indirect
	golang.org/x/net v0.0.0-20200114155413-6afb5195e5aa
	golang.org/x/sys v0.0.0-20200122134326-e047566fdf82
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
	google.golang.org/genproto v0.0.0-20200117163144-32f20d992d24 // indirect
	google.golang.org/grpc v1.26.0
	gopkg.in/ini.v1 v1.51.1 // indirect
	gopkg.in/src-d/go-git.v4 v4.13.1
	gopkg.in/yaml.v2 v2.2.7
	sigs.k8s.io/yaml v1.1.0 // indirect
)
