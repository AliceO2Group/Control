module github.com/AliceO2Group/Control

go 1.12

require (
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6
	github.com/armon/go-metrics v0.0.0-20190430140413-ec5e00d3c878
	github.com/beorn7/perks v1.0.1
	github.com/briandowns/spinner v0.0.0-20190611041244-e3fb08e7443c
	github.com/coreos/bbolt v1.3.3 // indirect
	github.com/coreos/etcd v3.3.10+incompatible
	github.com/coreos/go-semver v0.2.0
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/emirpasic/gods v1.12.0
	github.com/fatih/color v1.7.0
	github.com/gdamore/tcell v1.1.2
	github.com/gobwas/glob v0.0.0-20180208210656-5ccd90ef52e1
	github.com/gogo/protobuf v1.2.1
	github.com/golang/groupcache v0.0.0-20191227052852-215e87163ea7 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/btree v1.0.0 // indirect
	github.com/google/uuid v0.0.0-20190227210549-0cd6bf5da1e1
	github.com/gorilla/websocket v1.4.1 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.12.1 // indirect
	github.com/hashicorp/consul/api v1.1.0
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/memberlist v0.1.4 // indirect
	github.com/hpcloud/tail v1.0.0
	github.com/jinzhu/copier v0.0.0-20190625015134-976e0346caa8
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/json-iterator/go v1.1.6
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/k0kubun/pp v0.0.0-20190402025839-3d73dea227e0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/looplab/fsm v0.0.0-20190116160011-84b5307469f8
	github.com/mattn/go-colorable v0.1.2
	github.com/mattn/go-isatty v0.0.8
	github.com/mattn/go-runewidth v0.0.4
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/mesos/mesos-go v0.0.0-20190717023829-56ac038085ac
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b
	github.com/mitchellh/go-homedir v1.1.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/naoina/go-stringutil v0.0.0-20151118234443-6b638e95a32d
	github.com/naoina/toml v0.0.0-20170428090843-e6f5723bf2a6
	github.com/olekukonko/tablewriter v0.0.0-20181026071410-e6d60cf7ba1f
	github.com/onsi/ginkgo v1.6.0
	github.com/onsi/gomega v1.4.2
	github.com/pborman/uuid v0.0.0-20180906182336-adf5a7427709
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/prometheus/common v0.6.0
	github.com/prometheus/procfs v0.0.3
	github.com/rivo/tview v0.0.0-20190721135419-23dc8a0944e4
	github.com/rivo/uniseg v0.0.0-20190706090656-f8f8f751c732 // indirect
	github.com/russross/blackfriday v1.5.2
	github.com/sirupsen/logrus v1.4.2
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/spf13/cobra v0.0.0-20190607144823-f2b07da1e2c3
	github.com/spf13/pflag v1.0.3
	github.com/spf13/viper v1.3.2
	github.com/src-d/gcfg v1.4.0
	github.com/teo/logrus-prefixed-formatter v0.0.0-20171201112440-d4c78d981295
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/x-cray/logrus-prefixed-formatter v0.5.2 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77
	go.etcd.io/bbolt v1.3.3 // indirect
	go.uber.org/zap v1.13.0 // indirect
	golang.org/x/crypto v0.0.0-20190701094942-4def268fd1a4
	golang.org/x/net v0.0.0-20191002035440-2ec189313ef0
	golang.org/x/sys v0.0.0-20190804053845-51ab0e2deafa
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/grpc v1.24.0
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/src-d/go-git.v4 v4.0.0-20190801152248-0d1a009cbb60
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
	gopkg.in/warnings.v0 v0.1.2
	gopkg.in/yaml.v2 v2.2.3
)
