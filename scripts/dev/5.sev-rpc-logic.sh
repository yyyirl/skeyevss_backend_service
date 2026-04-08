#!/bin/bash

# api生成
source ./constants.sh

# 服务名称
server_name="db" # TODO
server_name_tmp=$(get_specific_parameter "-sev-name" "$@")
if [ -n "$server_name_tmp" ]; then
    server_name=$server_name_tmp
fi

# rpc service name {{.ServiceName}} TODO
service_name="deviceservice"
# 模块单数 {{.ServiceModuleNameSingular}} TODO
service_module_name_singular="Cascade"
# 模块复数 {{.ServiceModuleNamePlural}} TODO
service_module_name_plural="Cascade"
# model name {{.ModelName}} TODO
model_name="cascade"

if [[ ! -n "$server_name" ]]; then
    exitPrintln "项目名称不能为空"
    exit 1
fi

service_name_tmp=$(get_specific_parameter "-module" "$@")
if [ -n "$service_name_tmp" ]; then
    service_name=$(lowercase $service_name_tmp)
fi

service_module_name_singular_tmp=$(get_specific_parameter "-name" "$@")
if [ -n "$service_module_name_singular_tmp" ]; then
    service_module_name_singular=$(toPascalCase $service_module_name_singular_tmp)
fi

service_module_name_plural_tmp=$(get_specific_parameter "-names" "$@")
if [ -n "$service_module_name_plural_tmp" ]; then
    service_module_name_plural=$(toPascalCase $service_module_name_plural_tmp)
fi

model_name_tmp=$(get_specific_parameter "-model-name" "$@")
if [ -n "$model_name_tmp" ]; then
    model_name=$(lowercase $model_name_tmp)
fi

work_path=$SERVER_RPC_PATH/${server_name}/internal/logic/$service_name
mkdir -p $work_path

cd "${work_path}"
exitState "${work_path} 路径不存在"

singular=$(lowercase $service_module_name_singular)
plural=$(lowercase $service_module_name_plural)

\cp $SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH/create_logic.go.tpl $work_path/${singular}_create_logic.go
\cp $SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH/delete_logic.go.tpl $work_path/${singular}_delete_logic.go
\cp $SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH/update_logic.go.tpl $work_path/${singular}_update_logic.go
\cp $SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH/list_logic.go.tpl $work_path/${plural}_logic.go
\cp $SERVER_RPC_LOGIC_TEMPLATE_CUSTOM_PATH/row_logic.go.tpl $work_path/${singular}_row_logic.go

cd $work_path
ls -1 | while read item; do
    if [ ! -d "$item" ]; then
        if [ "$(uname)" == "Darwin" ]; then
            sed -i '' "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i '' "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i '' "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i '' "s|{{.ModelName}}|${model_name}|g" $item
        else
            sed -i "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i "s|{{.ModelName}}|${model_name}|g" $item
        fi
    fi
done

$FORMATTER -rm-unused -set-alias -format "$work_path/${singular}_create_logic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/${singular}_delete_logic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/${singular}_update_logic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/${plural}_logic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/${singular}_row_logic.go"
