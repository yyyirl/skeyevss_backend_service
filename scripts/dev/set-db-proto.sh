#!/bin/bash

source ./constants.sh

proto_path=$(get_specific_parameter "-path" "$@")
model_singular_name=$(get_specific_parameter "-name" "$@")
model_plural_name=$(get_specific_parameter "-names" "$@")
model_name_zh=$(get_specific_parameter "-name-zh" "$@")
sev_module=$(get_specific_parameter "-sev" "$@")

if [ -z "$proto_path" ]; then
    echo "-path 不能为空"
    exit 1
fi

if [ -z "$model_singular_name" ]; then
    echo "-name 不能为空"
    exit 1
fi

if [ -z "$model_plural_name" ]; then
    echo "-names 不能为空"
    exit 1
fi

if [ -z "$model_name_zh" ]; then
    echo "-name-zh 不能为空"
    exit 1
fi

if [ -z "$sev_module" ]; then
    echo "-sev 不能为空"
    exit 1
fi

# 转换方法名为首字母大写
singular_pascal=$(toPascalCase "$model_singular_name")
plural_pascal=$(toPascalCase "$model_plural_name")

# RPC 方法名列表
rpc_method_names=(
    "${singular_pascal}Create"
    "${singular_pascal}Delete"
    "${singular_pascal}Update"
    "${singular_pascal}Row"
    "${plural_pascal}"
)

# 方法对应的注释和完整定义
get_method_comment() {
    case "$1" in
        "${singular_pascal}Create")
            echo "// ${model_name_zh}创建"
            ;;
        "${singular_pascal}Delete")
            echo "// ${model_name_zh}删除"
            ;;
        "${singular_pascal}Update")
            echo "// ${model_name_zh}修改"
            ;;
        "${singular_pascal}Row")
            echo "// ${model_name_zh}详情"
            ;;
        "${plural_pascal}")
            echo "// ${model_name_zh}列表"
            ;;
    esac
}

get_method_rpc() {
    case "$1" in
        "${singular_pascal}Create")
            echo "rpc ${singular_pascal}Create(MapReq) returns (Response);"
            ;;
        "${singular_pascal}Delete")
            echo "rpc ${singular_pascal}Delete(XRequestParams) returns (Response);"
            ;;
        "${singular_pascal}Update")
            echo "rpc ${singular_pascal}Update(XRequestParams) returns (Response);"
            ;;
        "${singular_pascal}Row")
            echo "rpc ${singular_pascal}Row(IDReq) returns (Response);"
            ;;
        "${plural_pascal}")
            echo "rpc ${plural_pascal}(XRequestParams) returns (Response);"
            ;;
    esac
}

# 检查 proto 文件是否存在
if [ ! -f "$proto_path" ]; then
    echo "❌ proto 文件不存在: $proto_path"
    exit 1
fi

# 检查 sev_module 服务是否存在
if ! grep -q "service ${sev_module}[[:space:]]*{" "$proto_path"; then
    echo "❌ 服务 ${sev_module} 不存在于 proto 文件中"
    exit 1
fi

# 检查哪些方法已存在
missing_methods=()
for method_name in "${rpc_method_names[@]}"; do
    if ! grep -q "rpc ${method_name}[[:space:]]*(" "$proto_path"; then
        missing_methods+=("$method_name")
    fi
done

# 如果所有方法都存在，直接退出
if [ ${#missing_methods[@]} -eq 0 ]; then
    echo "⏭️  所有 RPC 方法已存在，跳过添加"
    exit 0
fi

# 创建临时文件
tmp_file=$(mktemp)

# 标记是否已添加任何方法
added_any=false
in_target_service=false

# 读取文件并处理
while IFS= read -r line; do
    # 检测是否进入目标服务
    if [[ "$line" =~ service[[:space:]]+${sev_module}[[:space:]]*\{ ]]; then
        in_target_service=true
        echo "$line"
        continue
    fi

    # 如果在目标服务内，检测服务结束
    if [ "$in_target_service" = true ]; then
        # 检测服务结束的 }
        if [[ "$line" =~ ^[[:space:]]*\} ]]; then
            # 在服务结束前添加缺失的方法
            if [ "$added_any" = false ] && [ ${#missing_methods[@]} -gt 0 ]; then
                echo ""
                # 按顺序添加缺失的方法
                for method_name in "${rpc_method_names[@]}"; do
                    # 检查这个方法是否缺失
                    is_missing=false
                    for missing in "${missing_methods[@]}"; do
                        if [ "$missing" = "$method_name" ]; then
                            is_missing=true
                            break
                        fi
                    done

                    # 如果方法缺失，则添加
                    if [ "$is_missing" = true ]; then
                        comment=$(get_method_comment "$method_name")
                        rpc_line=$(get_method_rpc "$method_name")
                        echo "    $comment"
                        echo "    $rpc_line"
                        added_any=true
                    fi
                done
            fi
            echo "$line"
            in_target_service=false
            continue
        fi
    fi

    echo "$line"
done < "$proto_path" > "$tmp_file"

# 检查是否添加成功
if [ "$added_any" = true ]; then
    # 替换原文件
    mv "$tmp_file" "$proto_path"
    echo "✅ RPC 方法已添加到服务 ${sev_module}"
else
    echo "❌ 添加失败"
    rm -f "$tmp_file"
    exit 1
fi