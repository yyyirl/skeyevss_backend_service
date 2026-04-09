#!/bin/bash

source ./constants.sh

permissions_backend_file="$MAIN_PATH/core/common/source/permissions/backend.go"
permissions_frontend_file="$MAIN_PATH/core/common/source/permissions/frontend.go"
model_name_zh=$(get_specific_parameter "-name-zh" "$@")
server_name=$(get_specific_parameter "-server-name" "$@")
backend_permissions_id=''
frontend_permissions_id=''
api_group=$(get_specific_parameter "-api-group" "$@")
api_group_item=$(get_specific_parameter "-api-group-item" "$@")
if [ -n "$api_group" ]; then
    api_group="${api_group}/"
fi

if [ -z "$model_name_zh" ]; then
    echo "-name-zh 不能为空"
    exit 1
fi

if [ -z "$server_name" ]; then
    echo "-server-name 不能为空"
    exit 1
fi

# 检查文件是否存在
if [ ! -f "$permissions_backend_file" ]; then
    echo "❌ 文件不存在: $permissions_backend_file"
    exit 1
fi

if [ ! -f "$permissions_frontend_file" ]; then
    echo "❌ 文件不存在: $permissions_frontend_file"
    exit 1
fi

# 获取最大编号的函数
get_max_number() {
    local file=$1
    local prefix=$2
    grep -oE "${prefix}_[0-9]+" "$file" | grep -oE '[0-9]+' | sort -n | tail -1
}

# 处理权限文件的函数
process_permission_file() {
    local file=$1
    local prefix=$2
    local num=$3
    local name=$4

    tmp_file=$(mktemp)

    awk -v prefix="$prefix" -v num="$num" -v name="$name" '
        BEGIN {
            const_added = 0
            item_added = 0
            in_prev = 0
            brace_count = 0
        }

        # 在 const 块结束时添加新常量
        !const_added && /^[[:space:]]*\)/ {
            print ""
            print "\t" prefix "_" num "     IdType = \"" prefix "_" num "\""
            print "\t" prefix "_" num "_1   IdType = \"" prefix "_" num "_1\""
            print "\t" prefix "_" num "_2   IdType = \"" prefix "_" num "_2\""
            print "\t" prefix "_" num "_3   IdType = \"" prefix "_" num "_3\""
            print "\t" prefix "_" num "_4   IdType = \"" prefix "_" num "_4\""
            print "\t" prefix "_" num "_5   IdType = \"" prefix "_" num "_5\""
            const_added = 1
            print
            next
        }

        $0 ~ "UniqueId: " prefix "_" (num-1) "," && !item_added {
            print
            in_prev = 1
            brace_count = 0
            next
        }

        in_prev && !item_added {
            if ($0 ~ /{/) brace_count++
            if ($0 ~ /}/) brace_count--

            print

            if (brace_count == 0 && $0 ~ /},/) {
                print "\t\t},"
                print "\t\t{"
                print "\t\t\tUniqueId: " prefix "_" num ","
                print "\t\t\tName:     \"" name "\","
                print "\t\t\tChildren: []*Item{"
                print "\t\t\t\t{"
                print "\t\t\t\t\tUniqueId: " prefix "_" num "_1,"
                print "\t\t\t\t\tName:     \"" name "列表\","
                print "\t\t\t\t},"
                print "\t\t\t\t{"
                print "\t\t\t\t\tUniqueId: " prefix "_" num "_2,"
                print "\t\t\t\t\tName:     \"" name "详情\","
                print "\t\t\t\t},"
                print "\t\t\t\t{"
                print "\t\t\t\t\tUniqueId: " prefix "_" num "_3,"
                print "\t\t\t\t\tName:     \"更新" name "\","
                print "\t\t\t\t},"
                print "\t\t\t\t{"
                print "\t\t\t\t\tUniqueId: " prefix "_" num "_4,"
                print "\t\t\t\t\tName:     \"删除" name "\","
                print "\t\t\t\t},"
                print "\t\t\t\t{"
                print "\t\t\t\t\tUniqueId: " prefix "_" num "_5,"
                print "\t\t\t\t\tName:     \"添加" name "\","
                print "\t\t\t\t},"
                print "\t\t\t},"
                print "\t\t"
                item_added = 1
                in_prev = 0
            }
            next
        }

        { print }
    ' "$file" > "$tmp_file"

    if [ $? -eq 0 ] && [ -s "$tmp_file" ]; then
        mv "$tmp_file" "$file"
        return 0
    else
        rm -f "$tmp_file"
        return 1
    fi
}

# 替换权限的函数
replace_permission() {
    local file=$1
    local permission=$2

    if [ ! -f "$file" ]; then
        echo "⚠️  文件不存在: $file"
        return 1
    fi

    sed -i '' "s/\"permissions\.TODO\"/permissions.${permission}/g" "$file"

    echo "✅ 已替换: $file -> permissions.${permission}"
}

# 获取 backend.go 最大编号
max_num_backend=$(get_max_number "$permissions_backend_file" "P_0")
if [ -z "$max_num_backend" ]; then
    echo "❌ 无法获取 backend.go 最大编号"
    exit 1
fi

next_num_backend=$((max_num_backend + 1))

if grep -q "P_0_${next_num_backend}" "$permissions_backend_file"; then
    echo "⏭️  P_0_${next_num_backend} 已存在，跳过添加"
else
    if process_permission_file "$permissions_backend_file" "P_0" "$next_num_backend" "$model_name_zh"; then
        echo "✅ backend.go 权限已添加 (P_0_${next_num_backend})"
        backend_permissions_id=P_0_${next_num_backend}
    else
        echo "❌ backend.go 添加失败"
        exit 1
    fi
fi

# 获取 frontend.go 最大编号
max_num_frontend=$(get_max_number "$permissions_frontend_file" "P_1")
if [ -z "$max_num_frontend" ]; then
    echo "❌ 无法获取 frontend.go 最大编号"
    exit 1
fi

next_num_frontend=$((max_num_frontend + 1))
if grep -q "P_1_${next_num_frontend}" "$permissions_frontend_file"; then
    echo "⏭️  P_1_${next_num_frontend} 已存在，跳过添加"
else
    if process_permission_file "$permissions_frontend_file" "P_1" "$next_num_frontend" "$model_name_zh"; then
        echo "✅ frontend.go 权限已添加 (P_1_${next_num_frontend})"
        frontend_permissions_id=P_1_${next_num_frontend}
    else
        echo "❌ frontend.go 添加失败"
        exit 1
    fi
fi

# 格式化代码
if [ -n "$FORMATTER" ]; then
    $FORMATTER -rm-unused -set-alias -format "$permissions_backend_file"
    $FORMATTER -rm-unused -set-alias -format "$permissions_frontend_file"
fi

# 替换权限
work_path="${SERVER_REST_PATH}/${server_name}/internal/handler/${api_group}${api_group_item}"
replace_permission "${work_path}/listhandler.go" "P_0_${next_num_backend}_1"
replace_permission "${work_path}/rowhandler.go" "P_0_${next_num_backend}_2"
replace_permission "${work_path}/updatehandler.go" "P_0_${next_num_backend}_3"
replace_permission "${work_path}/deletehandler.go" "P_0_${next_num_backend}_4"
replace_permission "${work_path}/createhandler.go" "P_0_${next_num_backend}_5"

$FORMATTER -rm-unused -set-alias -format "${work_path}/listhandler.go"
$FORMATTER -rm-unused -set-alias -format "${work_path}/rowhandler.go"
$FORMATTER -rm-unused -set-alias -format "${work_path}/updatehandler.go"
$FORMATTER -rm-unused -set-alias -format "${work_path}/deletehandler.go"
$FORMATTER -rm-unused -set-alias -format "${work_path}/createhandler.go"

echo "backend_permissions_id=${backend_permissions_id}"
echo "frontend_permissions_id=${frontend_permissions_id}"