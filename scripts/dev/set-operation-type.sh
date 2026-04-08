#!/bin/bash

source ./constants.sh

system_operation_logs_file="$MAIN_PATH/core/repositories/models/system-operation-logs/data.go"
model_singular_name=$(get_specific_parameter "-name" "$@")
model_name_zh=$(get_specific_parameter "-name-zh" "$@")

if [ -z "$model_singular_name" ]; then
    echo "-name 不能为空"
    exit 1
fi

if [ -z "$model_name_zh" ]; then
    echo "-name-zh 不能为空"
    exit 1
fi

name=$(toPascalCase "$model_singular_name")

# 检查文件是否存在
if [ ! -f "$system_operation_logs_file" ]; then
    echo "❌ 文件不存在: $system_operation_logs_file"
    exit 1
fi

# 创建临时文件
tmp_file=$(mktemp)

# 使用 awk 处理，分别检测每个项
awk -v name="$name" \
    -v zh="$model_name_zh" '
    BEGIN {
        # 定义需要添加的项（保持原始格式，包含逗号）
        constants_arr[1] = "Type" name "Create"
        constants_arr[2] = "Type" name "Update"
        constants_arr[3] = "Type" name "Delete"

        types_arr[1] = "Type" name "Create: Type" name "Create,"
        types_arr[2] = "Type" name "Update: Type" name "Update,"
        types_arr[3] = "Type" name "Delete: Type" name "Delete,"

        views_arr[1] = "Type" name "Create: \"" zh "创建\","
        views_arr[2] = "Type" name "Update: \"" zh "更新\","
        views_arr[3] = "Type" name "Delete: \"" zh "删除\","

        # 标记哪些已存在
        for (i in constants_arr) {
            constants_exist[constants_arr[i]] = 0
        }
        for (i in types_arr) {
            split(types_arr[i], tmp, ":")
            types_exist[tmp[1]] = 0
        }
        for (i in views_arr) {
            split(views_arr[i], tmp, ":")
            views_exist[tmp[1]] = 0
        }

        constants_added = 0
        types_added = 0
        views_added = 0
    }

    # 检查 constants 是否存在
    {
        for (const in constants_exist) {
            if ($0 ~ "^[[:space:]]*" const "[[:space:]]*(//.*)?$") {
                constants_exist[const] = 1
            }
        }
    }

    # 检查 types 是否存在
    {
        for (typ in types_exist) {
            if ($0 ~ "^[[:space:]]*" typ ":") {
                types_exist[typ] = 1
            }
        }
    }

    # 检查 views 是否存在
    {
        for (view in views_exist) {
            if ($0 ~ "^[[:space:]]*" view ":") {
                views_exist[view] = 1
            }
        }
    }

    # 在 baseMax 上一行添加缺失的 constants
    /^[[:space:]]*baseMax[[:space:]]*=/ && !constants_added {
        for (i in constants_arr) {
            if (constants_exist[constants_arr[i]] == 0) {
                printf "\t%s\n", constants_arr[i]
                constants_added = 1
            }
        }
        if (constants_added == 1) {
            printf "\n"
        }
        print
        next
    }

    # 在 Types 的 Known 上一行添加缺失的 types
    /^[[:space:]]*Known:[[:space:]]*Known,/ && !types_added {
        for (i in types_arr) {
            if (types_exist[types_arr[i]] == 0) {
                printf "\t\t\t%s\n", types_arr[i]
                types_added = 1
            }
        }
        if (types_added == 1) {
            printf "\n"
        }
        print
        next
    }

    # 在 TypeViews 的 Known 上一行添加缺失的 views
    /^[[:space:]]*Known:[[:space:]]*"未知类型",/ && !views_added {
        for (i in views_arr) {
            if (views_exist[views_arr[i]] == 0) {
                printf "\t\t\t%s\n", views_arr[i]
                views_added = 1
            }
        }
        if (views_added == 1) {
            printf "\n"
        }
        print
        next
    }

    { print }
' "$system_operation_logs_file" > "$tmp_file"

# 检查是否添加成功
if [ $? -eq 0 ] && [ -s "$tmp_file" ]; then
    # 替换原文件
    mv "$tmp_file" "$system_operation_logs_file"

    # 格式化代码
    if [ -n "$FORMATTER" ]; then
        $FORMATTER -rm-unused -set-alias -format "$system_operation_logs_file"
    fi
else
    echo "❌ 添加失败"
    rm -f "$tmp_file"
    exit 1
fi