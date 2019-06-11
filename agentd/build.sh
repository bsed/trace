#!/bin/bash


if [ "$1" = "" ];
then
    echo "发布环境不得为空,使用帮助 : ./build.sh mac"
    exit 0
fi 

# 生产agentd可执行文件
if [ "$1" = "linux" ];
then
GOOS=linux GOARCH=amd64 go build -o apmAgentd
echo "linux agent编译成功"
else
go build -o apmAgentd
echo "mac agent编译成功"
fi

