SKEYEVSS_TZ=Asia/Shanghai

######################################################### docker config
SKEYEVSS_CODE_PATH_HOST=code
SKEYEVSS_NETWORKS_DRIVER=bridge
# 基础镜像
SKEYEVSS_SEV_BASE_IMAGE=skeyevss_sev_base
SKEYEVSS_SEV_DEPENDS_FILE_IMAGE=skeyevss_sev_dependent_files
SKEYEVSS_SEV_JAEGER_IMAGE=skeyevss_sev_jaeger
SKEYEVSS_SEV_MYSQL_IMAGE=skeyevss_sev_mysql
SKEYEVSS_SEV_REDIS_IMAGE=skeyevss_sev_redis
SKEYEVSS_SEV_ETCD_IMAGE=skeyevss_sev_etcd
# 应用依赖文件挂载目录
SKEYEVSS_SEV_VOLUMES_DIR=/opt/skeyevss
# harbor
SKEYEVSS_HARBOR_DOMAIN="harbor.xxx.com"
SKEYEVSS_HARBOR_PORT="12100"

# 主域名
SKEYEVSS_DOMAIN="showcase.xxx.cn"

# Prometheus
SKEYEVSS_PROMETHEUS_PORT=10102
# Grafana
SKEYEVSS_GRAFANA_PORT=4000

# minio api域名 内网
SKEYEVSS_MINIO_API_TARGET=http://12.12.12.12:12401

# docker
SKEYEVSS_OS_ENVIRONMENT=docker

###################################ls###################### 链路追踪
# Jaeger
SKEYEVSS_JAEGER_NAME=SkeyevssSevJaeger
# Jaeger UI
SKEYEVSS_JAEGER_UI_PORT=16686
# Collector API
SKEYEVSS_JAEGER_COLLECTOR_API_PORT=14268
# Collector gRPC
SKEYEVSS_JAEGER_COLLECTOR_GRPC_PORT=14250
# Agent UDP
SKEYEVSS_JAEGER_AGENT_UDP1_PORT=6831
# Agent UDP
SKEYEVSS_JAEGER_AGENT_UDP2_PORT=6832
# Configs
SKEYEVSS_JAEGER_CONFIGS_PORT=5778
# OTLP gRPC
SKEYEVSS_JAEGER_OTLP_GRPC_PORT=4317
# OTLP HTTP
SKEYEVSS_JAEGER_OTLP_HTTP_PORT=4318
# 链路追踪
SKEYEVSS_TELEMETRY_ENDPOINT=http://___INTERNAL_IP___:14268/api/traces
SKEYEVSS_TELEMETRY_SAMPLER=0.1
SKEYEVSS_TELEMETRY_BATCHER=jaeger

# DTM HTTP
SKEYEVSS_DTM_HTTP_PORT=10104
# DTM gRPC
SKEYEVSS_DTM_GRPC_PORT=10105

######################################################### application config
# ssl 证书公钥路径 绝对路径
SKEYEVSS_SSL_CERT_PUBLIC_KEY=""
# ssl 证书私钥 绝对路径
SKEYEVSS_SSL_CERT_PRIVATE_KEY=""
SKEYEVSS_CONTAINERIZED_STATE=false
SKEYEVSS_PRODUCT_NAME=Skeyevss
SKEYEVSS_VERSION=V1.0.6
SKEYEVSS_ACTIVATE_CODE_FILENAME=.activate.code
SKEYEVSS_ACTIVATE_CODE_PATH=/app/etc/.activate.code
# 本机ip启动时候更新
SKEYEVSS_SEV_BIND_IP=0.0.0.0
# 宿主机内网ip
SKEYEVSS_INTERNAL_IP=___INTERNAL_IP___
# 宿主机公网ip
SKEYEVSS_EXTERNAL_IP=___EXTERNAL_IP___
# 环境 [dev pro]
SKEYEVSS_SERVER_ENV_MODE=pro
# 应用路径
SKEYEVSS_ROOT=/app
# 是否启用打包中的mysql
SKEYEVSS_ENABLED_MYSQL=true
# 是否启用打包中的reids
SKEYEVSS_ENABLED_REDIS=true
# 是否启用ffmpeg
SKEYEVSS_ENABLED_FFMPEG=true
# 是否启用etcd
SKEYEVSS_ENABLED_ETCD=true
# 是否启用密码初始化更新
SKEYEVSS_ENABLED_UPDATE_BACKEND_MANAGE_PWD=true
# 是否启用链路追踪禁用选项
SKEYEVSS_ENABLED_TELEMETRY_DISABLED=true
# 数据库类型 mysql,sqlite
SKEYEVSS_DATABASE_TYPE=sqlite
# 文件存储路径
SKEYEVSS_SAVE_FILE_DIR=source
# 保存运行sql路径 不填写则输出到控制台
SKEYEVSS_SAVE_SQL_DIR=/app/source/sql
# 设备录像保存目录
SKEYEVSS_SAVE_VIDEO_DIR=/app/source/video-records
# 视频快照保存目录
SKEYEVSS_SAVE_VIDEO_SNAPSHOT_DIR=/app/source/video-snapshot
# pprof文件存储目录
SKEYEVSS_SAVE_PPROF_DIR=/app/source/pprof

######################################################### log config
# 路径
SKEYEVSS_SERVER_LOG_PATH=/app/logs/applications
# [console file]
SKEYEVSS_LOG_MODE=file
# [json plain]
SKEYEVSS_LOG_ENCODING=plain
# [info error]
SKEYEVSS_LOG_LEVEL=error

######################################################### rpc拦截器
# 是否开启重试
SKEYEVSS_USE_RPC_CALLER_RETRY=false
# 重试次数
SKEYEVSS_RPC_CALLER_RETRY_MAX=3
# 重试等待时间 单位/毫秒
SKEYEVSS_RPC_CALLER_RETRY_WAIT_INTERVAL=100
# 是否启用keepalive
SKEYEVSS_USE_RPC_KEEPALIVE=false
# 发送 keepalive 探测的时间间隔 单位/s
SKEYEVSS_RPC_KEEPALIVE_TIME=10
# 等待响应超时时间 单位/s
SKEYEVSS_RPC_KEEPALIVE_TIMEOUT=5
# 即使没有活跃的流也发送 keepalive
SKEYEVSS_RPC_KEEPALIVE_PERMIT_WITHOUT_STREAM=false

######################################################### 管理后台账号信息
SKEYEVSS_BACKEND_ADMIN_USERNAME=admin
SKEYEVSS_BACKEND_ADMIN_PASSWORD=111111
SKEYEVSS_BACKEND_SUPER_USERNAME=super
SKEYEVSS_BACKEND_SUPER_PASSWORD=111111
SKEYEVSS_BACKEND_SHOWCASE_USERNAME=showcase
SKEYEVSS_BACKEND_SHOWCASE_PASSWORD=111111
SKEYEVSS_BACKEND_USE_SHOWCASE=false

######################################################### key
# openssl rand -base64 16 | tr -d '/+=' | cut -c1-16
SKEYEVSS_AES_KEY=G5PzheeawSwRl3fL
# openssl rand -hex 64
SKEYEVSS_U_KEY_BACKEND_API=
SKEYEVSS_U_KEY_MEDIA_SERVER=
SKEYEVSS_U_KEY_VSS=
SKEYEVSS_U_KEY_CRON=
SKEYEVSS_U_KEY_DB=
SKEYEVSS_U_KEY_WEB_SEV=
SKEYEVSS_U_KEY_SK_BACKEND_API=
SKEYEVSS_U_KEY_JWT=
# 天地图key
SKEYEVSS_TMAP_KEY=

######################################################### mysql
SKEYEVSS_MYSQL_PORT=11001
SKEYEVSS_MYSQL_HOST=___INTERNAL_IP___
SKEYEVSS_MYSQL_NAME=SkeyevssSevMysql
SKEYEVSS_MYSQL_USERNAME=root
SKEYEVSS_MYSQL_PASSWORD=
SKEYEVSS_MYSQL_DB_NAME_BASE=skeyevss

######################################################### sqlite
SKEYEVSS_SQLITE_DB_FILE=/app/data/sqlite/skeyevss.db

######################################################### ffmpeg
SKEYEVSS_FFMPEG_HOME=/usr/bin
SKEYEVSS_FFMPEG_NAME=SkeyevssSevFFMpeg
SKEYEVSS_FFMPEG_PORT=11017

######################################################### redis
SKEYEVSS_REDIS_PORT=11002
SKEYEVSS_REDIS_HOST=___INTERNAL_IP___
SKEYEVSS_REDIS_NAME=SkeyevssSevRedis
SKEYEVSS_REDIS_PASSWORD=

######################################################### etcd
SKEYEVSS_ETCD_HOST=___INTERNAL_IP___
# 节点间通信端口Peer Port
SKEYEVSS_ETCD_PEER_PORT=11016
# 客户端通信端口Client Port
SKEYEVSS_ETCD_CLIENT_PORT=11003
SKEYEVSS_ETCD_NAME=SkeyevssSevEtcd

######################################################### email
SKEYEVSS_EMAIL_HOST=
#SKEYEVSS_EMAIL_PORT=465
SKEYEVSS_EMAIL_PORT=
SKEYEVSS_EMAIL_USERNAME=
SKEYEVSS_EMAIL_PASSWORD=
SKEYEVSS_EMAILS=

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ web 代理服务器
SKEYEVSS_WEB_SEV_PORT=11004
SKEYEVSS_WEB_SEV_NAME=SkeyevssWebServer
SKEYEVSS_WEB_SEV_PROXY_API_BACKEND=/api-backend
SKEYEVSS_WEB_SEV_PROXY_API_EXTERNAL=/api-external
SKEYEVSS_WEB_SEV_PROXY_FILE=/x-assets
SKEYEVSS_WEB_SEV_PROXY_FILE_URL=http://___INTERNAL_IP___:11004/x-assets
SKEYEVSS_WEB_SEV_PROXY_FILE_URL_FRONTEND=https://___MAIN_HOST___/x-assets
SKEYEVSS_WEB_SEV_PROXY_API_EXTERNAL_TARGET=___INTERNAL_IP___
SKEYEVSS_WEB_SEV_CONF=/app/etc/.web-sev.yaml
SKEYEVSS_BACKEND_BUILD_NAME=SkeyevssWebBackend

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ media server
SKEYEVSS_MEDIA_SERVER_PORT=11005
SKEYEVSS_MEDIA_SERVER_NAME=SkeyevssSevMediaServer
SKEYEVSS_MEDIA_SERVER_CONF=/app/etc/.media-server.yaml
SKEYEVSS_MEDIA_SERVER_CONF_DEF=/app/etc/skeyesms.conf
SKEYEVSS_MEDIA_SERVER_HTTPS_PORT=14443
SKEYEVSS_MEDIA_SERVER_RTSP_PORT=15544
SKEYEVSS_MEDIA_SERVER_RTMP_PORT=19350
SKEYEVSS_MEDIA_SERVER_RTC_PORT_MIN=14888
SKEYEVSS_MEDIA_SERVER_RTC_PORT_MAX=14908
# RTMP开始推流通知(当前做为上级,下级(设备)给当前推流)
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_PUB_START=http://___INTERNAL_IP___:11013/api/notify/on-pub-start
# RTMP停止推流通知(当前做为上级,下级(设备)给当前推流)
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_PUB_STOP=http://___INTERNAL_IP___:11013/api/notify/on-pub-stop
# RTMP开始推流通知(当前作为下级(设备),给上级推流)
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_PUSH_START=http://___INTERNAL_IP___:11013/api/notify/on-push-start
# RTMP停止推流通知(当前作为下级(设备),给上级推流)
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_PUSH_STOP=http://___INTERNAL_IP___:11013/api/notify/on-push-stop
# 开始拉流通知
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_RELAY_PULL_START=http://___INTERNAL_IP___:11013/api/notify/on-reply-pull-start
# 停止拉流通知
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_RELAY_PULL_STOP=http://___INTERNAL_IP___:11013/api/notify/on-reply-pull-stop
# 有RTMP推流连接建立的事件通知
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_RTMP_CONNECT=http://___INTERNAL_IP___:11013/api/notify/on-rtmp-connect
# 开始播放通知
SKEYEVSS_MEDIA_SERVER_NOTIFY_ON_SUB_START=http://___INTERNAL_IP___:11013/api/notify/on-sub-start
# rtc数据发送绑定ip 多个地址使用, 分隔
SKEYEVSS_MEDIA_RTC_ICE_HOST_NAT_TO_IPS=""

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ vss
# 服务器 sse event 代理地址
SKEYEVSS_VSS_SSE_TARGET_FRONTEND=https://___MAIN_HOST___/sse-event
SKEYEVSS_VSS_HTTP_TARGET_FRONTEND=https://___MAIN_HOST___
SKEYEVSS_VSS_USE_EXTERNAL_IP=false
SKEYEVSS_PRINT_REQUEST_LOG=true
# 保存sip log
SKEYEVSS_PRINT_SAVE_SIP_LOG_FILE=true
SKEYEVSS_VSS_PORT=11008
SKEYEVSS_VSS_HTTP_PORT=11013
SKEYEVSS_VSS_SSE_PORT=11014
SKEYEVSS_VSS_INVITE_USABLE_MIN_PORT=13000
SKEYEVSS_VSS_INVITE_USABLE_MAX_PORT=13500
SKEYEVSS_VSS_BIND_HOST=0.0.0.0
SKEYEVSS_VSS_NAME=SkeyevssSevVss
SKEYEVSS_VSS_CONF=/app/etc/.vss.yaml
SKEYEVSS_VSS_SIP_USE_PASSWORD=true
SKEYEVSS_VSS_SIP_ID=31010000042220000002
SKEYEVSS_VSS_SIP_DOMAIN=3101000004
SKEYEVSS_VSS_SIP_PASSWORD=12345678
# 定时发送catalog请求周期
SKEYEVSS_VSS_SIP_CATALOG_INTERVAL=60
# 心跳超时时间
SKEYEVSS_VSS_SIP_LIFETIME_TIMEOUT_INTERVAL=180
# 向设备发送请求超时时间单位/s
SKEYEVSS_VSS_SIP_SEND_TIMEOUT_INTERVAL=5
# 接收推流超时时间
SKEYEVSS_VSS_MEDIA_RECEIVE_TIMEOUT_INTERVAL=60
# 无人观看流关闭时间
SKEYEVSS_VSS_MEDIA_NO_WATCHING_TIMEOUT_INTERVAL=20
# media server创建接收推流端口最大值
SKEYEVSS_VSS_MEDIA_SERVER_STREAM_PORT_MAX=15000
# media server创建接收推流端口最小值
SKEYEVSS_VSS_MEDIA_SERVER_STREAM_PORT_MIN=19000
# 是否开启公网流量接受
SKEYEVSS_VSS_SIP_USE_EXTERNAL_WAN=false
# gbs和media server是否处在同一台机器
SKEYEVSS_VSS_SIP_MEDIA_SERVER_VSS_SAME_MACHINE=true
# onvif 多播地址
SKEYEVSS_VSS_ONVIF_MULTI_CAST_IP=239.255.255.250
# onvif 多播端口
SKEYEVSS_VSS_ONVIF_WS_DISCOVERY_PORT=3702
# onvif discover超时时间 单位/s
SKEYEVSS_VSS_ONVIF_DISCOVERY_TIMEOUT=3
# 国标级联
SKEYEVSS_VSS_CASCADE_SIP_PORT=11015
# websocket
SKEYEVSS_VSS_WS_PORT=11018
# 最大链接数量
SKEYEVSS_VSS_WS_MAX_CONN=300000
# read 最大读取 单位/bytes
SKEYEVSS_VSS_WS_READ_BUFFER_MAX_SIZE=102400
# write 最大发送 单位/bytes
SKEYEVSS_VSS_WS_WRITEBUFFER_MAX_SIZE=102400
# 超时时间 长时间无动作 单位/s
SKEYEVSS_VSS_WS_WAIT_TIME_OUT=20
# 心跳间隔
SKEYEVSS_VSS_WS_HEARTBEAT_TIMER=1
# 过期清理语音对讲sip周期 单位/s
SKEYEVSS_VSS_WS_CLEAR_TALK_SIP_INTERVAL=15000
# 以下代理地址必须结合SKEYEVSS_DOMAIN设置了才会生效
# 播放地址代理
SKEYEVSS_VSS_STREAM_PLAY_PROXY_WS="sms-play-ws"
SKEYEVSS_VSS_STREAM_PLAY_PROXY_HTTP="sms-play-http"
# websocket代理
SKEYEVSS_WS_PROX="ws"

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ 任务
SKEYEVSS_CRON_PORT=11009
SKEYEVSS_CRON_NAME=SkeyevssSevCron
SKEYEVSS_CRON_CONF=/app/etc/.cron.yaml

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ rpc db
SKEYEVSS_DB_GRPC_NAME=SkeyevssSevDB
SKEYEVSS_DB_GRPC_PORT=11010
SKEYEVSS_DB_GRPC_CONF=/app/etc/.db-rpc.yaml

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ backend-api
#SKEYEVSS_BACKEND_API=backend-api
SKEYEVSS_BACKEND_API_NAME=SkeyevssSevBackendApi
SKEYEVSS_BACKEND_API_PORT=11011
SKEYEVSS_BACKEND_API_CONF=/app/etc/.backend-api.yaml
# 健康检查路径
SKEYEVSS_DEV_SERVER_HEALTH_PATH=/ping

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ guard server
SKEYEVSS_GUARD_NAME=SkeyevssSevGuard
SKEYEVSS_GUARD_CONF=/app/etc/.guard.yaml
SKEYEVSS_GUARD_PORT=11012

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ sk-backend-api 官网后台
SKEYEVSS_SK_BACKEND_API_NAME=SkeyevssSevSKBackendApi
SKEYEVSS_SK_BACKEND_API_PORT=11030
SKEYEVSS_SK_BACKEND_API_CONF=/app/etc/.sk-backend-api.yaml

# application @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ sk-backend-api 官网前台
SKEYEVSS_SK_FRONTEND_API_NAME=SkeyevssSevSKFrontendApi
SKEYEVSS_SK_FRONTEND_API_PORT=11031
SKEYEVSS_SK_FRONTEND_API_CONF=/app/etc/.sk-frontend-api.yaml

######################################################### 应用构建
# 主应用开始所在目录
SKEYEVSS_SEV_RES_ASSET_DIR=skeyevss-sev
SKEYEVSS_SEV_RES_REDIS_ASSET=skeyevss-sev/redis
SKEYEVSS_SEV_RES_MYSQL_ASSET=skeyevss-sev/mysql
SKEYEVSS_SEV_RES_ETCD_ASSET=skeyevss-sev/etcd
SKEYEVSS_SEV_RES_FFMPEG_ASSET=skeyevss-sev/ffmpeg
SKEYEVSS_SEV_RES_BACKEND_WEB_ASSET=skeyevss-sev/backend-web
SKEYEVSS_SEV_RES_APP_SEV_ASSET=skeyevss-sev/sev
SKEYEVSS_SEV_RES_ETC_ASSET=skeyevss-sev/sev/etc
SKEYEVSS_SEV_RES_ASSET_SCRIPTS_DIR=skeyevss-sev/scripts

# 数据目录
SKEYEVSS_SEV_RES_ASSET_CERT_DIR=
SKEYEVSS_SEV_RES_ASSET_DATA_DIR=skeyevss-sev/data
SKEYEVSS_SEV_RES_ASSET_DATA_LOG=skeyevss-sev/logs
SKEYEVSS_SEV_RES_MYSQL_DATA=skeyevss-sev/data/mysql
SKEYEVSS_SEV_RES_REDIS_DATA=skeyevss-sev/data/redis
SKEYEVSS_SEV_RES_ETCD_DATA=skeyevss-sev/data/etcd
SKEYEVSS_SEV_RES_API_DIR=skeyevss-sev/sev/doc/api

# 存放依赖服务配置文件
SKEYEVSS_SEV_RES_CONFIG_DIR=skeyevss-sev/etc
SKEYEVSS_SEV_RES_MYSQL_CONFIG_PATH=skeyevss-sev/mysql/my.ini
SKEYEVSS_SEV_RES_REDIS_CONFIG_PATH=skeyevss-sev/etc/redis.ini
SKEYEVSS_SEV_RES_ETCD_CONFIG_PATH=skeyevss-sev/etc/etcd.ini

# 前端代码路径(管理后台)
SKEYEVSS_BACKEND_WEB_CODE_PATH=/Users/yiyiyi/Code/web/skeyevss_backend
# media server
SKEYEVSS_MEDIA_SERVER_CODE_PATH=/Users/yiyiyi/Code/golang/src/skeyesms
# GOPATH
SKEYEVSS_GOPATH=/Users/yiyiyi/Code/golang

######################################################### pprof
SKEYEVSS_PPROF_BACKEND_API_PORT=11020
SKEYEVSS_PPROF_DB_RPC_PORT=11021
SKEYEVSS_PPROF_VSS_PORT=11022
SKEYEVSS_PPROF_WEB_PORT=11023
SKEYEVSS_PPROF_CRON_PORT=11024

######################################################### gen
# 国标id生成 前缀 平台
SKEYEVSS_GEN_PLATFORM_UNIQUEID=34020000002000000000
# 国标id生成 前缀 目录
SKEYEVSS_GEN_DIR_UNIQUEID=34020000002160000000
# 国标id生成 前缀 nvr
SKEYEVSS_GEN_NVR_UNIQUEID=34020000001110000000
# 国标id生成 前缀 摄像机
SKEYEVSS_GEN_CAMERA_UNIQUEID=34020000001320000000