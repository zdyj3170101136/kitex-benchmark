/*
 * Copyright 2021 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"

	grpcg "github.com/cloudwego/kitex-benchmark/codec/protobuf/grpc_gen"
	"github.com/cloudwego/kitex-benchmark/perf"
	"github.com/cloudwego/kitex-benchmark/runner"
	"github.com/felixge/fgprof"
	"google.golang.org/grpc"
)

const (
	port = 8000
)

var recorder = perf.NewRecorder("GRPC@Server")

type server struct {
	grpcg.UnimplementedEchoServer
}

func (s *server) Echo(ctx context.Context, req *grpcg.Request) (*grpcg.Response, error) {
	resp := runner.ProcessRequest(recorder, req.Action, req.Msg)

	return &grpcg.Response{
		Msg:    resp.Msg,
		Action: resp.Action,
	}, nil
}

var (
	memProfile   *string
	cpuProfile   *string
	blockProfile *string
	mutexProfile *string
	traceFile    *string
	fgProfile    *string
)

func main() {
	memProfile = flag.String("test.memprofile", "", "write an allocation profile to `file`")
	cpuProfile = flag.String("test.cpuprofile", "", "write a cpu profile to `file`")
	blockProfile = flag.String("test.blockprofile", "", "write a goroutine blocking profile to `file`")
	mutexProfile = flag.String("test.mutexprofile", "", "write a mutex contention profile to the named file after execution")
	traceFile = flag.String("test.trace", "", "write an execution trace to `file`")
	fgProfile = flag.String("test.fgprofile", "", "write an fg profile to `file`")
	flag.Parse()
	if *memProfile != "" {
		f, _ := os.Create(*memProfile)
		p := pprof.Lookup("heap")
		defer p.WriteTo(f, 0)
	}
	if *cpuProfile != "" {
		f, _ := os.Create(*cpuProfile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *traceFile != "" {
		f, _ := os.Create(*traceFile)
		trace.Start(f)
		defer trace.Stop()
	}
	if *blockProfile != "" {
		f, _ := os.Create(*blockProfile)
		runtime.SetBlockProfileRate(1)
		p := pprof.Lookup("block")
		defer p.WriteTo(f, 0)
	}
	if *mutexProfile != "" {
		f, _ := os.Create(*mutexProfile)
		runtime.SetMutexProfileFraction(1)
		p := pprof.Lookup("mutex")
		defer p.WriteTo(f, 0)
	}
	if *fgProfile != "" {
		f, _ := os.Create(*fgProfile)
		cancel := fgprof.Start(f, fgprof.FormatPprof)
		defer func() {
			err := cancel()
			if err != nil {
				log.Fatal(err)
			}
		}()
	}
	// start pprof server
	go func() {
		perf.ServeMonitor(fmt.Sprintf(":%d", port+10000))
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	grpcg.RegisterEchoServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())

	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
