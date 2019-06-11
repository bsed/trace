#!/bin/bash

# 删除过往的mecury和taitan、pinpoint
pkill -9 mecury
pkill -9 taitan-client
rm -rf /httx/run/mecury
rm -rf /httx/run/taitan
rm -rf /httx/run/openapm-agent/

# 删除过往的agentd
pkill -9 apm-agent 
pkill -9 apmAgentd
rm -rf /httx/run/apm-agentd 


# 创建目录
mkdir -p /httx/run/apm-agentd && cd /httx/run/apm-agentd

# 下载agentd和conf
wget -O apmAgentd http://web-apmWeb-vip/web/agentd/download/apmAgentd
wget -O agentd.conf http://web-apmWeb-vip/web/agentd/download/agentd.conf 

chmod +x ./apmAgentd

nohup ./apmAgentd 1>out.log 2>error.log &

