#!/bin/bash
set -e
CURDIR=$(cd $(dirname $0); pwd)

echo "Checking whether the environment meets the requirements ..."
source $CURDIR/env.sh
echo "Check finished."

#numactl --hardware
#available: 1 nodes (0)
#node 0 cpus: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15
#node 0 size: 31726 MB
#node 0 free: 22854 MB
#node distances:
#node   0
#  0:  10
cmd_server="nice -n -20 numactl -C 0,1,2,3 -m 0"
cmd_client="nice -n -20 numactl -C 4-15 -m 0"

srepo=("grpc")
crepo=("grpc")
ports=(8000 8006)

echo "Building grpc services by exec build_grpc.sh..."
source $CURDIR/build_grpc.sh
echo "Build finished."

cmds=(
""
"-test.cpuprofile=cpu.pprof"
"-test.trace=trace.pprof"
"-test.blockprofile=block.pprof"
"-test.mutexprofile=mutex.pprof"
"-test.cpuprofile=cpu.pprof -test.memprofile=mem.pprof -test.trace=trace.pprof -test.blockprofile=block.pprof -test.mutexprofile=mutex.pprof"
"-test.memprofile=mem.pprof"
"-test.fgprofile=fg.pprof"
)

# benchmark
for b in ${body[@]}; do
  for c in ${concurrent[@]}; do
    for ((i = 0; i < ${#srepo[@]}; i++)); do
      srp=${srepo[i]}
      crp=${crepo[i]}
      addr="127.0.0.1:${ports[i]}"

      for ((j = 0; j < ${#cmds[@]}; j++)) do
        # server start
              echo "Starting server [$srp], if failed please check [output/log/nohup.log] for detail."
              server="${cmd_server} ${output_dir}/bin/${srp}_reciever ${cmds[j]} >> ${output_dir}/log/nohup.log 2>&1"
              echo "${server}"

              nohup $server &
              sleep 1

              # run client
             client="${cmd_client} ${output_dir}/bin/${crp}_bencher -addr="${addr}" -b=${b} ${cmds[j]} -c=${c} -n=${n} --sleep=${sleep}"
             echo "${client}"
             $client

              # stop server
              pid=$(ps -ef | grep ${srp}_reciever | grep -v grep | awk '{print $2}')
              kill -9 $pid
              echo "Server [$srp] stopped, pid [$pid]."
              sleep 1
      done

        # server start
        server="perf record -F 100 -g -B -C 0-3 ${cmd_server} ${output_dir}/bin/${srp}_reciever >> ${output_dir}/log/nohup.log 2>&1"
        echo "${server}"
        nohup $server &
        sleep 1

        # run client
        client="perf record -F 100 -g -B -C 4-15 ${cmd_client} ${output_dir}/bin/${crp}_bencher -addr="${addr}" -b=${b} -c=${c} -n=${n} --sleep=${sleep}"
        echo "${client}"
        $client

         # stop server
        pid=$(ps -ef | grep ${srp}_reciever | grep -v grep | awk '{print $2}')
        kill -9 $pid
        echo "Server [$srp] stopped, pid [$pid]."
        sleep 1
    done
  done
done

finish_cmd
