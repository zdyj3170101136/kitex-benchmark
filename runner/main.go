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

package runner

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"

	"github.com/cloudwego/kitex-benchmark/perf"
	"github.com/felixge/fgprof"
)

var (
	address    string
	echoSize   int
	total      int64
	concurrent int
	poolSize   int
	sleepTime  int
)

type Options struct {
	Address  string
	Body     []byte
	PoolSize int
}

type ClientNewer func(opt *Options) Client

type Client interface {
	Echo(action, msg string) (err error)
}

type Response struct {
	Action string
	Msg    string
}

var (
	memProfile   *string
	cpuProfile   *string
	blockProfile *string
	mutexProfile *string
	traceFile    *string
	fgProfile    *string
)

func initFlags() {
	fgProfile = flag.String("test.fgprofile", "", "write an fg profile to `file`")
	memProfile = flag.String("test.memprofile", "", "write an allocation profile to `file`")
	cpuProfile = flag.String("test.cpuprofile", "", "write a cpu profile to `file`")
	blockProfile = flag.String("test.blockprofile", "", "write a goroutine blocking profile to `file`")
	mutexProfile = flag.String("test.mutexprofile", "", "write a mutex contention profile to the named file after execution")
	traceFile = flag.String("test.trace", "", "write an execution trace to `file`")
	flag.StringVar(&address, "addr", "127.0.0.1:8000", "client call address")
	flag.IntVar(&echoSize, "b", 1024, "echo size once")
	flag.IntVar(&concurrent, "c", 100, "call concurrent")
	flag.Int64Var(&total, "n", 1024*100, "call total nums")
	flag.IntVar(&poolSize, "pool", 10, "conn poll size")
	flag.IntVar(&sleepTime, "sleep", 0, "sleep time for every request handler")
	flag.Parse()
}

func Main(name string, newer ClientNewer) {
	initFlags()

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
		err := perf.ServeMonitor(":18888")
		if err != nil {
			fmt.Printf("perf monitor server start failed: %v\n", err)
		} else {
			fmt.Printf("perf monitor server start success\n")
		}
	}()

	r := NewRunner()

	opt := &Options{
		Address:  address,
		PoolSize: poolSize,
	}
	cli := newer(opt)
	payload := string(make([]byte, echoSize))
	action := EchoAction
	if sleepTime > 0 {
		action = SleepAction
		st := strconv.Itoa(sleepTime)
		payload = fmt.Sprintf("%s,%s", st, payload[len(st)+1:])
	}
	handler := func() error { return cli.Echo(action, payload) }

	// === warming ===
	r.Warmup(handler, concurrent, 100*1000)

	// === beginning ===
	if err := cli.Echo(BeginAction, "empty"); err != nil {
		log.Fatalf("beginning server failed: %v", err)
	}
	recorder := perf.NewRecorder(fmt.Sprintf("%s@Client", name))
	recorder.Begin()

	// === benching ===
	r.Run(name, handler, concurrent, total, echoSize, sleepTime)

	// == ending ===
	recorder.End()
	if err := cli.Echo(EndAction, "empty"); err != nil {
		log.Fatalf("ending server failed: %v", err)
	}

	// === reporting ===
	recorder.Report() // report client
	fmt.Printf("\n\n")
}
