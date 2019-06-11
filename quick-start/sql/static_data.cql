--
-- 该文件存储的都是静态数据，例如用户数据、配置数据等，相对监控的链路数据来说更加重要，因此需要做replication
--

CREATE KEYSPACE tracing_static WITH replication = {'class': 'SimpleStrategy', 'replication_factor': '3'}  
    AND durable_writes = false;


USE tracing_static;

-- 用户表
CREATE TABLE IF NOT EXISTS account (
    id               text,
    name             text,
    mobile           text,
    email            text,
    message          text,
    app_show         tinyint, -- 1代表显示所有应用，2代表显示自己设定的应用
    app_names        text, -- 用户自定义显示的应用列表
    last_login_date  text,
    PRIMARY KEY (id)
) WITH gc_grace_seconds = 10800;

-- 登录次数统计表
CREATE TABLE IF NOT EXISTS login_count (
    id               text,
    count          counter,
    PRIMARY KEY (id)
) WITH gc_grace_seconds = 10800;

-- 管理员表
CREATE TABLE IF NOT EXISTS admin (
    id text,
    priv text,
    PRIMARY KEY (id)
) WITH gc_grace_seconds = 10800;

INSERT INTO admin (id,priv) VALUES ('13269','super_admin');
INSERT INTO admin (id,priv) VALUES ('14929','super_admin');

-- 告警用户组表
CREATE TABLE IF NOT EXISTS alerts_group (
    id               UUID,          -- 唯一ID
    name             text,         -- 用户组名称
    channel          text,         -- 告警通道，支持'mobile' 'email' 'message'
    owner            text,         -- 当前组的所有者ID
    users            list<text>,   -- 该组的组员
    update_date      bigint,         -- 记录更新时间
    PRIMARY KEY (id,owner)
) WITH gc_grace_seconds = 10800;

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_group (owner) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;


-- 告警策略模版中的监控项
CREATE TYPE alert (
    name text,                      -- 监控项名称
    type text,                      -- 监控项类型： apm、system
    label text,                     -- 监控项描述
    compare tinyint,                -- 比较类型 1: > 2:<  3:=
    unit text,                      -- 单位：%、个 
    duration tinyint,               -- 持续时间, 1 代表1分钟
    keys text,                      -- 为一些特殊指标使用，例如http code告警，此处就是code list
    value double,
);

-- 告警策略模版表
CREATE TABLE IF NOT EXISTS alerts_policy (
    id               UUID,                      -- 唯一ID
    name             text,                      -- 策略模版名
    owner            text,                      -- 当前策略模版的所有者ID
    alerts           list<frozen<alert>>,       -- 告警项列表
    update_date      bigint,                    -- 记录更新时间
    PRIMARY KEY (id,owner)
) WITH gc_grace_seconds = 10800;

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_policy (update_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_policy (owner) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;

-- 应用告警表
CREATE TABLE IF NOT EXISTS alerts_app (
    name             text,              -- 应用名
    owner            text,              -- 该应用告警的所有者ID
    policy_id        text,              -- 应用关联的告警策略模版
    group            text,              -- 应用关联的用户组
    channel          text,              -- 不使用用户组时，单独设置告警通道
    users            list<text>,        -- 不使用用户组时,单独设置通知的用户
    api_alerts      text,               -- 额外的api告警项设置(可以自行指定url，设置特殊的告警规则)
    update_date      bigint,            -- 记录更新时间
    PRIMARY KEY (name,owner)
) WITH gc_grace_seconds = 10800;

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_app (update_date) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' 
    WITH OPTIONS = {'mode': 'SPARSE'};

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_app (owner) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_app (policy_id) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;

CREATE CUSTOM INDEX IF NOT EXISTS ON alerts_app (group) 
    USING 'org.apache.cassandra.index.sasi.SASIIndex' ;



-- Method ID映射表
CREATE TABLE IF NOT EXISTS app_methods (
    app_name            text,
    method_id           int,
    method_info         text,
    line                int,
    type                int,
    PRIMARY KEY (app_name, method_id)
) WITH gc_grace_seconds = 10800;  -- 3 hours of downtime acceptable on nodes

-- Sql ID映射表
CREATE TABLE IF NOT EXISTS app_sqls (
    app_name            text,
    sql_id              int,
    sql_info            text,
    PRIMARY KEY (app_name, sql_id)
) WITH gc_grace_seconds = 10800;


-- Str ID映射表
CREATE TABLE IF NOT EXISTS app_strs (
    app_name            text,
    str_id              int,
    str_info            text,
    PRIMARY KEY (app_name, str_id)
)WITH gc_grace_seconds = 10800; 


CREATE TABLE IF NOT EXISTS agents (
    app_name            text,
    agent_id            text,
    service_type        int,
    socket_id           int,
    host_name           text,
    ip                  text,
    pid                 int,
    version             text,
    start_time          bigint,
    end_time            bigint,
    is_live             BOOLEAN,
    is_container        BOOLEAN,
    operating_env       int,
    tracing_addr        text,
    agent_info          text,
    PRIMARY KEY (app_name, agent_id)
) WITH gc_grace_seconds = 10800;


CREATE TABLE IF NOT EXISTS apps (
    app_name        text,
    -- last_count_time BIGINT,
    PRIMARY KEY (app_name)
) WITH gc_grace_seconds = 10800;  


-- app api表
CREATE TABLE IF NOT EXISTS  app_apis (
    app_name            text, -- app name
    api                 text, -- api信息
    api_type            int,  -- api Type
    PRIMARY KEY (app_name, api)
) WITH gc_grace_seconds = 10800;  

