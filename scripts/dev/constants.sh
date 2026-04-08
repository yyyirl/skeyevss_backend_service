#!/bin/bash

set -e

source ./functions.sh
exitState "functions.sh 导入失败!"

# 主项目路径
export MAIN_PATH=$(dirname "$(dirname "$(dirname "$(realpath "$0")")")")
# 项目名称
export PROJECT_DIR=$(basename "$MAIN_PATH")
# go env
#export GOPATH=$HOME/code/golang/src
#export GOBIN=$HOME/code/golang/bin

export MODULE_NAME=$(awk '/^module / {print $2}' $MAIN_PATH/go.mod)

# 后端服务代码路径
export SERVER_PATH=$MAIN_PATH
# restfulApi路径
export SERVER_REST_PATH=$SERVER_PATH/core/app/sev
export SERVER_SK_REST_PATH=$SERVER_PATH/core/app/sk
# rpc路径
export SERVER_RPC_PATH=$SERVER_PATH/core/app/sev
# 前端代码路径
export SERVER_FRONTEND_PATH=$MAIN_PATH/frontend
# 仓库地址
export SERVER_REPOSITORIES_PATH=$SERVER_PATH/core/repositories
# 代码模板文件地址
export SERVER_TEMPLATE_PATH=$SERVER_PATH/templates/template
# 自定义模板地址 mysql
export SERVER_MYSQL_TEMPLATE_CUSTOM_PATH=$SERVER_PATH/templates/template-custom/mysql
# 自定义模板地址
export SERVER_ES_TEMPLATE_CUSTOM_PATH=$SERVER_PATH/templates/template-custom/es
# 自定义模板地址 api logic
export SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH=$SERVER_PATH/templates/template-custom/api-logic
# 自定义模板地址 api handler
export SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH=$SERVER_PATH/templates/template-custom/handler
# 自定义模板地址 rpc logic
export SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH=$SERVER_PATH/templates/template-custom/rpc-logic

# 可执行文件路径
export SKEYEVSS_BIN=$MAIN_PATH/bin
export GO=/usr/local/go/bin/go
export PATH=$PATH:$GOBIN:$SKEYEVSS_BIN

# 工具
export GO_CONTROL=goctl.exe
if [ "$(uname)" == "Darwin" ]; then
#    GO_CONTROL=goctl.1.8.3
#    chmod +x $GOBIN/$GO_CONTROL
    GO_CONTROL=goctl
fi

if [ "$(uname)" != "Darwin" ]; then
    GO=go.exe
fi

if ! command -v $GO_CONTROL &> /dev/null; then
    $GO install github.com/zeromicro/go-zero/tools/goctl@latest
fi

# 配置文件信息
println "cyan" "项目路径: $MAIN_PATH"
println "cyan" "goctl版本: $($GO_CONTROL -v)"

# 安装go文件格式化
FORMATTER=goimports-reviser
if [ "$(uname)" != "Darwin" ]; then
    FORMATTER=${FORMATTER}.exe
fi

#if [[ ! -f "${FORMATTER}" ]]; then
#    $GO install github.com/incu6us/goimports-reviser/v3@latest
#fi

# 检查goctl
$GO_CONTROL env check --verbose --install
