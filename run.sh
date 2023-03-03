#!/bin/zsh

# 当触发ctrl当时候，推出，删除当前文件夹下面当server文件，并且向当前进程阻中所有进程发送信号，让停止
trap "rm server; kill 0" SIGINT

go build -o server
./server -port=8001 &
./server -port=8002 &
./server -port=8003 -api=1 &

sleep 2
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &
curl "http://localhost:9999/api?key=Tom" &

# 下面这条命令仍然会重新请求，因为之前当请求已经删除了，如果g.m不delete当话，可以续上来，但是可能后面数据更新了，不准确
sleep 3
echo ">>> start test"
curl "http://localhost:9999/api?key=Tom" &

wait