#!/bin/bash

clear

source ./constants.sh

rpc_sev_client='Backend'
rpc_sev_module='BackendService'
model_name_zh='用户'
model_name='user'
model_names='users'

# 生成model
echo -e "开始生成model ------------------ \n"
bash ./1.sev-db.sh -name "$model_names"

# 添加model到service_context.go
echo
echo -e "开始设置model ------------------ \n"
bash ./set-db-model.sh -name "$model_names" -name-zh "$model_name_zh"

# 添加api内容
echo
echo -e "开始添加api ------------------ \n"
bash ./set-api.sh -path ${MAIN_PATH}/templates/apis/backend-api.api -name $model_name -names $model_names -name-zh $model_name_zh

# 设置api
echo
echo -e "开始设置api ------------------ \n"
bash ./2.sev-api.sh -name "backend"

# 设置rpc proto
echo
echo -e "开始设置proto ------------------ \n"
bash ./set-db-proto.sh -path ${MAIN_PATH}/core/app/sev/db/db.proto -name $model_name -names $model_names -name-zh $model_name_zh -sev $rpc_sev_module

# 生成rpc代码
echo
echo -e "开始生成rpc代码 ------------------ \n"
bash ./3.sev-rpc.sh -name "db"

# 设置rpc logic
echo
echo -e "开始设置rpc logic ------------------ \n"
bash ./5.sev-rpc-logic.sh -sev-name "db" -module $rpc_sev_module -name $model_name -names $model_names -model-name $model_names

# 设置api logic
echo
echo -e "开始设置api logic ------------------ \n"
bash ./4.sev-api-rpc-logic.sh -sev-name "db" -module $rpc_sev_module -name $model_name -names $model_names -model-name $model_names -service-client $rpc_sev_client

# 设置日志类型
echo
echo -e "开始设置日志类型 ------------------ \n"
bash ./set-operation-type.sh -name $model_name -name-zh $model_name_zh

# 设置权限
echo
echo -e "开始设置权限 ------------------ \n"
bash ./set-permissions.sh -name-zh $model_name_zh -name $model_names -server-name "backend"

echo ""
echo "✅ ${model_names} 模块创建完成"
echo "  TODO: 数据表创建 autoMigrate: core/app/sev/db/internal/svc/init_database.go"
echo ""