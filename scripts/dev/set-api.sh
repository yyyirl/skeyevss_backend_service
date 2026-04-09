#!/bin/bash

source ./constants.sh

api_path=$(get_specific_parameter "-path" "$@")
model_singular_name=$(get_specific_parameter "-name" "$@")
model_plural_name=$(get_specific_parameter "-names" "$@")
model_name_zh=$(get_specific_parameter "-name-zh" "$@")
api_group=$(get_specific_parameter "-api-group" "$@")
api_group_item=$(get_specific_parameter "-api-group-item" "$@")

if [ -z "$api_path" ]; then
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

if [ -n "$api_group" ]; then
    api_group="${api_group}/"
fi

# 定义要添加的内容
content_to_add=$(cat << EOF

// ${model_name_zh}
@server (
	group:      ${api_group}${api_group_item}
	middleware: AuthMiddleware
)
service backend-api {
	@doc "创建${model_name_zh}"
	@handler Create
	post /${model_plural_name} (RecordReq)

	@doc "更新${model_name_zh}"
	@handler Update
	put /${model_singular_name} (TRFParams)

	@doc "${model_name_zh}列表"
	@handler List
	post /${model_singular_name}/list (TRFParams)

	@doc "${model_name_zh}详情"
	@handler Row
	get /${model_singular_name}/:id (IdQuery)

	@doc "删除"
	@handler Delete
	delete /${model_plural_name} (TRFParams)
}
EOF
)

# 检查文件是否存在
if [ -f "$api_path" ]; then
    # 检查是否已存在 Create handler（或其他唯一标识）
    if grep -q "@handler Create" "$api_path" && grep -q "post /${model_plural_name}" "$api_path"; then
        echo "⏭️  内容已存在，跳过添加: $api_path"
        exit 0
    fi
else
    echo "❌ $api_path 文件不存在"
fi

# 追加内容到文件
echo "$content_to_add" >> "$api_path"

if [ $? -eq 0 ]; then
    echo "✅ 内容已添加到: $api_path"
else
    echo "❌ 添加失败"
    exit 1
fi