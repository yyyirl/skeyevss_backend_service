#!/bin/bash

# api生成
source ./constants.sh

# 模块名称
server_name="db" # TODO 模块名
server_name_tmp=$(get_specific_parameter "-name" "$@")
if [ -n "$server_name_tmp" ]; then
    server_name=$server_name_tmp
fi

if [[ ! -n "$server_name" ]]; then
    exitPrintln "项目名称不能为空"
    exit 1
fi

work_path=$SERVER_RPC_PATH/$server_name
mkdir -p $work_path

cd "${work_path}"
exitState "${work_path} 路径不存在"

println "blue" "work path: [$(pwd)]"
# 生成rpc
rm -rf $work_path/proto
cp -r $SERVER_PATH/templates/proto $work_path/proto

if [ "$(uname)" == "Darwin" ]; then
    sed -i '' 's|google/protobuf/any.proto|proto/google/protobuf/any.proto|g' $work_path/$server_name.proto
    sed -i '' 's|google/protobuf/struct.proto|proto/google/protobuf/struct.proto|g' $work_path/$server_name.proto
else
    sed -i 's|google/protobuf/any.proto|proto/google/protobuf/any.proto|g' $work_path/$server_name.proto
    sed -i 's|google/protobuf/struct.proto|proto/google/protobuf/struct.proto|g' $work_path/$server_name.proto
fi

println "blue" "生成命令: $GO_CONTROL --style go_zero rpc protoc $server_name.proto --go_out=. --go-grpc_out=. --zrpc_out=. -m"
$GO_CONTROL --style go_zero rpc protoc $server_name.proto --go_out=. --go-grpc_out=. --zrpc_out=. -m
if [ "$(uname)" == "Darwin" ]; then
    sed -i '' 's|proto/google/protobuf/any.proto|google/protobuf/any.proto|g' $work_path/$server_name.proto
    sed -i '' 's|proto/google/protobuf/struct.proto|google/protobuf/struct.proto|g' $work_path/$server_name.proto
else
    sed -i 's|proto/google/protobuf/any.proto|google/protobuf/any.proto|g' $work_path/$server_name.proto
    sed -i 's|proto/google/protobuf/struct.proto|google/protobuf/struct.proto|g' $work_path/$server_name.proto
fi

exitState "rpc生成失败"

rm -rf $work_path/proto

# 配置文件
yaml="${server_name}-rpc.yaml"
if [[ ! -f "${SERVER_PATH}/etc/.${yaml}" ]]; then
    mv ${work_path}/etc/${server_name}.yaml ${SERVER_PATH}/etc/.${yaml}
fi
rm -rf ${work_path}/etc

# 移动入口文件
if [[ ! -f "${work_path}/main.go" ]]; then
    mv "${work_path}/${server_name}.go" "${work_path}/main.go"
else
    rm -rf "${work_path}/${server_name}.go"
fi
