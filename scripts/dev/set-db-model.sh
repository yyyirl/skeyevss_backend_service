#!/bin/bash

source ./constants.sh

model_name=$(get_specific_parameter "-name" "$@")
model_name_zh=$(get_specific_parameter "-name-zh" "$@")
if [ -z "$model_name" ]; then
    echo "-name 不能为空 "
    exit 1
fi

if [ -z "$model_name_zh" ]; then
    echo "-name-zh 不能为空 "
    exit 1
fi

model_name_1=$(toPascalCase "$model_name")

# 生成model
 bash ./1.sev-db.sh -name "$model_name"

# 添加model到service_context.go
db_service_context_file="$MAIN_PATH/core/app/sev/db/internal/svc/service_context.go"

if grep -q "${model_name_1}Model" "$db_service_context_file"; then
    echo
else
    tmp_file=$(mktemp)
    awk -v module_name="$MODULE_NAME" \
        -v model_name="$model_name" \
        -v model_name_1="$model_name_1" \
        -v model_name_zh="$model_name_zh" '
        {
            # 添加 import - 在最后一个 import 行后添加
            if ($0 ~ /^[[:space:]]*"/ && !import_added && !in_block) {
                # 记录 import 行
                last_import_line = NR
                print
                next
            }

            # 在 import 块结束后添加新的 import
            if (!import_added && NR > last_import_line && last_import_line > 0 && $0 !~ /^[[:space:]]*"/ && $0 !~ /^[[:space:]]*$/) {
                printf "\t\"%s/core/repositories/models/%s\"\n", module_name, model_name
                import_added = 1
                print
                next
            }

            # 添加字段定义 - 在结构体最后一个字段后添加
            if ($0 ~ /^[[:space:]]+[A-Z][a-zA-Z0-9]+Model[[:space:]]+\*[a-z]+\.[A-Za-z]+/ && !field_added) {
                print
                # 记录这是最后一个字段
                last_field = $0
                next
            }

            # 在结构体结束前添加新字段
            if ($0 ~ /^}/ && !field_added && last_field != "") {
                printf "\t%sModel *%s.DB         // %s\n", model_name_1, model_name, model_name_zh
                field_added = 1
                print
                next
            }

            # 添加初始化 - 在 return 的最后一个初始化后添加
            if ($0 ~ /^[[:space:]]+[A-Z][a-zA-Z0-9]+Model:[[:space:]]+[a-z]+\.[A-Za-z]+\(dbClient\),/ && !init_added) {
                print
                # 记录这是最后一个初始化
                last_init = $0
                next
            }

            # 在 return 结束前添加新初始化
            if ($0 ~ /^[[:space:]]+}/ && !init_added && last_init != "") {
                printf "\t\t%sModel:         %s.NewDB(dbClient),\n", model_name_1, model_name
                init_added = 1
                print
                next
            }

            # 打印所有行
            print
        }
    ' "$db_service_context_file" > "$tmp_file"

    if [ -s "$tmp_file" ]; then
        mv "$tmp_file" "$db_service_context_file"
        echo "✅ ${model_name_1}Model 已成功添加"
        $FORMATTER -rm-unused -set-alias -format $db_service_context_file
    else
        echo "❌ ${model_name_1}Model 添加失败"
        rm -f "$tmp_file"
        exit 1
    fi
fi