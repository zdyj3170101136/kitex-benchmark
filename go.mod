module github.com/cloudwego/kitex-benchmark

go 1.15

require (
	github.com/apache/thrift v0.14.0
	github.com/cloudfoundry/gosigar v1.3.3
	github.com/cloudwego/fastpb v0.0.2
	github.com/cloudwego/kitex v0.4.3
	github.com/felixge/fgprof v0.9.3
	github.com/gogo/protobuf v1.3.2
	github.com/lesismal/arpc v1.2.4
	github.com/lesismal/nbio v1.1.23-0.20210815145206-b106d99bce56
	github.com/montanaflynn/stats v0.6.6
	github.com/shirou/gopsutil v3.21.11+incompatible
	github.com/smallnest/rpcx v1.6.11
	github.com/tklauser/go-sysconf v0.3.11 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	google.golang.org/grpc v1.42.0
	google.golang.org/protobuf v1.28.0
)

replace github.com/apache/thrift => github.com/apache/thrift v0.13.0
