#!/bin/bash

# api生成
source ./constants.sh

# TODO 模块名称
server_name="backend" # TODO
server_name_tmp=$(get_specific_parameter "-name" "$@")
if [ -n "$server_name_tmp" ]; then
    server_name=$server_name_tmp
fi

# 使用orm.params
use_orm_params="1" # TODO
api_name=${server_name}-api.api
work_path=${SERVER_REST_PATH}/${server_name}
yaml="${server_name}-api.yaml"

mkdir -p $work_path

cd "${work_path}"
exitState "${work_path} 路径不存在"

default_handler_file=$SERVER_TEMPLATE_PATH/api/handler.tpl
default_logic_file=$SERVER_TEMPLATE_PATH/api/logic.tpl
handler_file=$SERVER_MYSQL_TEMPLATE_CUSTOM_PATH/api-handler.tpl
logic_file=$SERVER_MYSQL_TEMPLATE_CUSTOM_PATH/api-logic.tpl
if [ $use_orm_params == "2" ]; then
    handler_file=$SERVER_MYSQL_TEMPLATE_CUSTOM_PATH/api-sp-handler.tpl
    logic_file=$SERVER_MYSQL_TEMPLATE_CUSTOM_PATH/api-sp-logic.tpl
fi

if [ $use_orm_params == "1" ] || [ $use_orm_params == "2" ]; then
    # 模板替换
    \cp $default_handler_file ${default_handler_file}.tmp
    \cp $default_logic_file ${default_logic_file}.tmp
    \cp $handler_file ${default_handler_file}
    \cp $logic_file ${default_logic_file}
fi

# 生成api
$GO_CONTROL api go --api ${SERVER_PATH}/templates/apis/${api_name} -dir . --home $SERVER_TEMPLATE_PATH
if [ $use_orm_params == "1" ] || [ $use_orm_params == "2" ]; then
    # 模板替换
    \mv ${default_handler_file}.tmp $default_handler_file
    \mv ${default_logic_file}.tmp $default_logic_file
fi
exitState "api生成失败"

# 配置文件
if [[ ! -f "${SERVER_PATH}/etc/.${yaml}" ]]; then
    mv ${work_path}/etc/${yaml} ${SERVER_PATH}/etc/.${yaml}
fi
rm -rf ${work_path}/etc

# 移动入口文件
if [[ ! -f "${work_path}/main.go" ]]; then
    mv "${work_path}/${server_name}.go" "${work_path}/main.go"
else
    rm -rf "${work_path}/${server_name}.go"
fi
#
## 特殊处理
#if [ $server_name == "frontend" ]; then
#   for file in "$work_path/internal/logic/finance"/*; do
#       if [ -f "$file" ]; then
#           filename=$(basename "$file")
#           if [[ "$filename" != *"+"* ]]; then
#               echo "Deleting file: $file"
#               rm -rf "$file"
#           fi
#       fi
#   done
#fi


#find "${work_path}/internal/handler" -name "*.go" | while read -r file; do
#    $FORMATTER -file-path $file -project-name hyper-sn -rm-unused
#done
#
#find "${work_path}/internal/logic" -name "*.go" | while read -r file; do
#    $FORMATTER -file-path $file -project-name hyper-sn -rm-unused
#done
