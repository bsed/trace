CREATE KEYSPACE tracing_data WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '1'}  
    AND durable_writes = false;

-- ALTER KEYSPACE tracing_data  
-- WITH replication =  {'class': 'SimpleStrategy', 'replication_factor': '1'}  AND durable_writes = false



USE tracing_data;


CREATE TABLE IF NOT EXISTS traces (
    trace_id            text,
    span_id             bigint,

    app_name            text,
    agent_id            text,

    duration            int,
    api                 text,
    service_type        int,
    end_point           text,
    remote_addr         text,

    annotations         BLOB,
    event_list          blob,
   
    parent_id           bigint,

    method_id           int,
    exception_info      blob,
    error               tinyint,   
         
    input_date          bigint,
    PRIMARY KEY (trace_id, span_id)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE TABLE IF NOT EXISTS  traces_index (
    app_name        text,
    agent_id        text,

    trace_id        text,
    span_id         bigint,
	
    api             text,
    remote_addr     text,
    input_date      bigint, 
    duration        int,

    error           tinyint,            
    PRIMARY KEY (app_name, input_date, trace_id)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE CUSTOM INDEX IF NOT EXISTS ON traces_index (api) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;


CREATE TABLE IF NOT EXISTS traces_chunk (
    trace_id            text,
    span_id             bigint,
    cid                 bigint,
   
    event_list     blob,
    PRIMARY KEY (trace_id, span_id, cid)
)WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;


-- agent runtime 信息表
CREATE TABLE IF NOT EXISTS agent_runtime (
    app_name            text,
    agent_id            text,
    runtime_type        int,
    input_date          bigint,
    metrics             blob,
    PRIMARY KEY (app_name, agent_id, input_date)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;


CREATE CUSTOM INDEX IF NOT EXISTS ON agent_runtime (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};



CREATE TABLE IF NOT EXISTS  api_stats (
    app_name                    text,               -- 目标应用
   
    -- 被其它应用访问数据
    count                       int,                -- 被访问次数
    err_count                   int,                -- 被访问错误数
    duration                    int,                -- 被访问总耗时
    max_duration                int,                -- 最大访问耗时
    min_duration                int,                -- 最小访问耗时
    satisfaction                int,               -- 满意
    tolerate                    int,                -- 可容忍
    api                         text,               -- 目标应用被访问的api
    input_date                  bigint,
    PRIMARY KEY (app_name, api, input_date)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE CUSTOM INDEX IF NOT EXISTS ON api_stats (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};



CREATE TABLE IF NOT EXISTS  exception_stats (
    app_name            text,
    method_id           int,
    class_id            INT,
    input_date          bigint,
    service_type        int,
    total_elapsed       int,
    max_elapsed         int,
    min_elapsed         int,
    count               int,
    PRIMARY KEY (app_name, method_id, class_id, input_date)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;


CREATE CUSTOM INDEX IF NOT EXISTS ON exception_stats (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};


CREATE TABLE IF NOT EXISTS method_stats (
    app_name        text,
    api             text,
    method_id       int,
    input_date      bigint,
    service_type    int,
    elapsed         int,
    max_elapsed     int,
    min_elapsed     int,
    count           int,
    err_count       int,
    PRIMARY KEY (app_name, api, input_date, method_id)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE CUSTOM INDEX IF NOT EXISTS ON method_stats (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};


CREATE TABLE IF NOT EXISTS sql_stats (
    app_name        text,
    sql             int,
    input_date      bigint,
    elapsed         int,
    max_elapsed     int,
    min_elapsed     int,
    count           int,
    err_count       int,
    PRIMARY KEY (app_name, sql, input_date)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE CUSTOM INDEX IF NOT EXISTS ON sql_stats (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};


-- api被应用调用统计表
CREATE TABLE IF NOT EXISTS api_map (
    source_name                text,              -- 源应用名
    source_type                int,               -- 源应用类型
    
    target_name                text,              -- 目标应用名  
    target_type                int,               -- 目标应用类型   

    access_count               int,               -- 访问总数
    access_err_count           int,               -- 访问错误数
    access_duration            int,               -- 访问总耗时
    
    api                        text,               -- api id
    input_date                 bigint,             -- 插入时间
    PRIMARY KEY (target_name, input_date, api, source_name)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;

CREATE CUSTOM INDEX IF NOT EXISTS ON api_map (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};

CREATE TABLE IF NOT EXISTS service_map (
    source_name                text,               -- 源应用名
    source_type                int,                -- 源应用类型
    target_name                text,               -- 目标应用名  
    target_type                int,                -- 目标应用类型    

    access_count               int,                -- 访问总数
    access_err_count           int,                -- 访问错误数
    access_duration            int,                -- 访问总耗时

    input_date                 bigint,             -- 插入时间
    PRIMARY KEY (source_name, target_name, input_date, target_type)
) WITH gc_grace_seconds = 10800  AND  default_time_to_live = 2592000;


CREATE CUSTOM INDEX IF NOT EXISTS ON service_map (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};

CREATE CUSTOM INDEX IF NOT EXISTS ON service_map (target_name) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex';


CREATE TYPE IF NOT EXISTS alert (
    name text,                      -- 监控项名称
    type text,                      -- 监控项类型： apm、system
    label text,                     -- 监控项描述
    compare tinyint,                -- 比较类型 1: > 2:<  3:=
    unit text,                      -- 单位：%、个 
    duration tinyint,               -- 持续时间, 1 代表1分钟
    keys text,                      -- 为一些特殊指标使用，例如http code告警，此处就是code list
    value double,
);

CREATE TABLE IF NOT EXISTS alert_history (
    const_id                tinyint,           -- 常数主键，排序用
    id                      bigint,            -- 唯一ID
    app_name                text,              -- 应用名
    type                    tinyint,           -- 类型, 1: 告警 2: 恢复
    agent_id                text,              -- agent id
    host_name               text,              -- 主机名
    api                     text,              -- 接口字符串
    sql                     int,               -- sql id
    alert                   alert,             -- 告警时的监控项配置
    alert_value             double,            -- 发生告警时数据
    channel                 text,              -- 告警通道
    users                   list<text>,        -- 通知用户列表
    input_date              bigint,            -- 告警时间
    PRIMARY KEY (const_id, id)                 
) WITH gc_grace_seconds = 10800 and  CLUSTERING ORDER BY(id DESC) and default_time_to_live = 2592000; 

CREATE CUSTOM INDEX IF NOT EXISTS ON alert_history (input_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};

CREATE CUSTOM INDEX IF NOT EXISTS ON alert_history (app_name) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex';

CREATE TABLE IF NOT EXISTS agentd_info (
    hostname                text,              -- 主机名
    start_time              bigint,            -- agentd启动时间
    download_times          int,               -- agent下载次数
    restart_times           int,               -- agent重启次数
    last_download_time      bigint,            -- 上次下载agent的时间
    last_restart_time       bigint,            -- 上次重启agent的时间
    agentd_version          text,              -- agentd自身版本号
    agent_version           text,              -- agent版本号
    download_version        text,              -- 下载的agent版本
    admin_port              text,              -- agentd admin端口
    input_date              bigint,            -- 记录更新时间
    PRIMARY KEY (hostname)                 
) WITH gc_grace_seconds = 10800;

