module github.com/lim-team/LiMaoIM

go 1.15

require (
	github.com/DataDog/zstd v1.4.8 // indirect
	github.com/RussellLuo/timingwheel v0.0.0-20201029015908-64de9d088c74
	github.com/VictoriaMetrics/metrics v1.17.2 // indirect
	github.com/armon/go-metrics v0.3.8 // indirect
	github.com/bwmarrin/snowflake v0.3.0
	github.com/cockroachdb/errors v1.8.4 // indirect
	github.com/cockroachdb/pebble v0.0.0-20210611193018-b0b80de37120 // indirect
	github.com/cockroachdb/redact v1.0.9 // indirect
	github.com/eapache/queue v1.1.0
	github.com/edsrzf/mmap-go v1.0.0
	github.com/gin-contrib/pprof v1.3.0
	github.com/gin-gonic/gin v1.7.2
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4
	github.com/hashicorp/memberlist v0.2.4 // indirect
	github.com/jinzhu/configor v1.2.1
	github.com/judwhite/go-svc v1.2.1
	github.com/klauspost/compress v1.13.0 // indirect
	github.com/kr/pretty v0.2.1 // indirect
	github.com/lni/dragonboat/v3 v3.3.4
	github.com/miekg/dns v1.1.42 // indirect
	github.com/panjf2000/ants/v2 v2.4.1
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.29.0 // indirect
	github.com/sendgrid/rest v2.6.4+incompatible
	github.com/stretchr/testify v1.7.0
	github.com/tangtaoit/go-metrics v1.0.1
	github.com/tangtaoit/limnet v0.0.0-20210420102023-06d3eb19a0cd
	go.etcd.io/bbolt v1.3.6
	go.etcd.io/etcd/server/v3 v3.5.0-rc.1
	go.uber.org/atomic v1.8.0
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a
	golang.org/x/exp v0.0.0-20210611181813-f956177b079c // indirect
	golang.org/x/net v0.0.0-20210614182718-04defd469f4e // indirect
	golang.org/x/sys v0.0.0-20210616094352-59db8d763f22 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba
	google.golang.org/protobuf v1.26.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace github.com/tangtaoit/limnet => /Users/tt/work/projects/limao/go/limnet
