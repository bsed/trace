#!/bin/bash

# 将agentd和conf文件打包成一个zip
# 用户将agentd.zip和版本号文件放在服务器下载目录(tracecat/web/download)下

if [ "$1" = "" ];
then
    echo "发布环境不得为空,使用帮助 : ./release.sh mac 2.0.0"
    exit 0
fi 


if [ "$2" = "" ];
then 
    echo "版本号不得为空, 使用帮助 : ./release.sh mac 2.0.0" 
    exit 0
fi

# 删除之前发布的zip包
rm -f release/agent.zip
rm -rf ../web/download/*

# 拷贝配置文件
mkdir -p release/apm-agent
cp agent.yaml release/apm-agent/

# 生产agentd可执行文件
if [ "$1" = "linux" ];
then
GOOS=linux GOARCH=amd64 go build -o release/apm-agent/apm-agent
echo "linux agent编译成功"
else
go build -o release/apm-agent/apm-agent
echo "mac agent编译成功"
fi

# 打包
zip -r agent.zip release/
mv agent.zip release/

# 删除临时文件
rm -rf release/apm-agent

# 拷贝到服务器目录下
cp release/agent.zip ../web/download/
echo $2 > ../web/download/version
