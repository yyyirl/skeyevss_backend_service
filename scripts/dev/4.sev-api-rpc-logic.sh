#!/bin/bash

# api生成
source ./constants.sh

# 服务名称 api项目
server_name="backend" # TODO

# rpc service name {{.ServiceName}} TODO
service_name="deviceservice"
# 模块名称 {{.ModuleName}} TODO
service_module_name="cascade"
# 模块单数 {{.ServiceModuleNameSingular}} TODO
service_module_name_singular="Cascade"
# 模块复数 {{.ServiceModuleNamePlural}} TODO
service_module_name_plural="Cascade"
# rpc service {{.ServiceClient}} TODO
service_client="Device"
# {{.LogType}} log type
log_type="Cascade"

service_name_tmp=$(get_specific_parameter "-module" "$@")
if [ -n "$service_name_tmp" ]; then
    service_name=$(lowercase $service_name_tmp)
fi

model_name_tmp=$(get_specific_parameter "-model-name" "$@")
if [ -n "$model_name_tmp" ]; then
    service_module_name=$(lowercase $model_name_tmp)
fi

service_module_name_singular_tmp=$(get_specific_parameter "-name" "$@")
if [ -n "$service_module_name_singular_tmp" ]; then
    service_module_name_singular=$(toPascalCase $service_module_name_singular_tmp)
    log_type=$service_module_name_singular
fi

service_module_name_plural_tmp=$(get_specific_parameter "-names" "$@")
if [ -n "$service_module_name_plural_tmp" ]; then
    service_module_name_plural=$(toPascalCase $service_module_name_plural_tmp)
fi

service_client_tmp=$(get_specific_parameter "-service-client" "$@")
if [ -n "$service_client_tmp" ]; then
    service_client=$(toPascalCase $service_client_tmp)
fi

if [[ ! -n "$server_name" ]]; then
    exitPrintln "项目名称不能为空"
    exit 1
fi

work_path=${SERVER_REST_PATH}/${server_name}/internal/logic/${service_module_name_plural}
mkdir -p $work_path

cd "${work_path}"
exitState "${work_path} 路径不存在"

\cp $SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH/createlogic.go.tpl $work_path/createlogic.go
\cp $SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH/deletelogic.go.tpl $work_path/deletelogic.go
\cp $SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH/updatelogic.go.tpl $work_path/updatelogic.go
\cp $SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH/listlogic.go.tpl $work_path/listlogic.go
\cp $SERVER_API_LOGIC_TEMPLATE_CUSTOM_PATH/rowlogic.go.tpl $work_path/rowlogic.go

cd $work_path
ls -1 | while read item; do
    if [ ! -d "$item" ]; then
        if [ "$(uname)" == "Darwin" ]; then
            sed -i '' "s|{{.ModelName}}|${service_module_name}|g" $item
            sed -i '' "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i '' "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i '' "s|{{.ServiceClient}}|${service_client}|g" $item
            sed -i '' "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i '' "s|{{.ModuleName}}|${service_module_name}|g" $item
            sed -i '' "s|{{.LogType}}|${log_type}|g" $item
        else
            sed -i "s|{{.ModelName}}|${service_module_name}|g" $item
            sed -i "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i "s|{{.ServiceClient}}|${service_client}|g" $item
            sed -i "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i "s|{{.ModuleName}}|${service_module_name}|g" $item
            sed -i "s|{{.LogType}}|${log_type}|g" $item
        fi
    fi
done

$FORMATTER -rm-unused -set-alias -format "$work_path/createlogic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/deletelogic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/updatelogic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/listlogic.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/rowlogic.go"

work_path=${SERVER_REST_PATH}/${server_name}/internal/handler/${service_module_name_plural}
mkdir -p $work_path

cd "${work_path}"
exitState "${work_path} 路径不存在"

\cp $SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH/createhandler.go.tpl $work_path/createhandler.go
\cp $SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH/deletehandler.go.tpl $work_path/deletehandler.go
\cp $SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH/updatehandler.go.tpl $work_path/updatehandler.go
\cp $SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH/listhandler.go.tpl $work_path/listhandler.go
\cp $SERVER_API_HANDLER_TEMPLATE_CUSTOM_PATH/rowhandler.go.tpl $work_path/rowhandler.go

cd $work_path
ls -1 | while read item; do
    if [ ! -d "$item" ]; then
        if [ "$(uname)" == "Darwin" ]; then
            sed -i '' "s|{{.ModelName}}|${service_module_name}|g" $item
            sed -i '' "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i '' "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i '' "s|{{.ServiceClient}}|${service_client}|g" $item
            sed -i '' "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i '' "s|{{.ModuleName}}|${service_module_name}|g" $item
            sed -i '' "s|{{.LogType}}|${log_type}|g" $item
        else
            sed -i "s|{{.ModelName}}|${service_module_name}|g" $item
            sed -i "s|{{.ServiceModuleNameSingular}}|${service_module_name_singular}|g" $item
            sed -i "s|{{.ServiceModuleNamePlural}}|${service_module_name_plural}|g" $item
            sed -i "s|{{.ServiceClient}}|${service_client}|g" $item
            sed -i "s|{{.ServiceName}}|${service_name}|g" $item
            sed -i "s|{{.ModuleName}}|${service_module_name}|g" $item
            sed -i "s|{{.LogType}}|${log_type}|g" $item
        fi
    fi
done

$FORMATTER -rm-unused -set-alias -format "$work_path/createhandler.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/deletehandler.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/updatehandler.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/listhandler.go"
$FORMATTER -rm-unused -set-alias -format "$work_path/rowhandler.go"

