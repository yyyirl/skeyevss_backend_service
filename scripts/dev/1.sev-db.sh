#!/bin/bash

# models module生成
source ./constants.sh

OLD_GOBIN=$GOBIN
export GOBIN=$SKEYEVSS_BIN

# 安装sql转换器
sql2gorm=$SKEYEVSS_BIN/sql2gorm
if [ "$(uname)" != "Darwin" ]; then
    sql2gorm=${sql2gorm}.exe
fi

if [[ ! -f "${sql2gorm}" ]]; then
    $GO install github.com/cascax/sql2gorm@latest
fi

export GOBIN=$OLD_GOBIN

# TODO 模块名
name="users"
name_tmp=$(get_specific_parameter "-name" "$@")
if [ -n "$name_tmp" ]; then
    name=$name_tmp
fi

# TODO 主键类型
#primary_key_type="string"

file_path=$SERVER_PATH/templates/sql/${name}.sql
file_path_tmp=$SERVER_PATH/templates/sql/${name}.sql.tmp
\cp $file_path $file_path_tmp
#sed -ri "" 's/DEFAULT\s+\1\(JSON_ARRAY\(\)\)/ /' $file_path

if [ "$(uname)" == "Darwin" ]; then
    sed -i "" 's/DEFAULT (JSON_ARRAY())//' $file_path
else
    sed -i 's/DEFAULT (JSON_ARRAY())//' $file_path
fi

if [ ! -f "${file_path}" ]; then
    exitPrintln "${file_path} 文件不存在"
    exit 1
fi

mkdir -p ${SERVER_REPOSITORIES_PATH}/models/${name}

# model type
model_type_file=${SERVER_REPOSITORIES_PATH}/models/${name}/model.go
rm -rf $model_type_file
# 表字段
model_columns_file=${SERVER_REPOSITORIES_PATH}/models/${name}/variables.go
rm -rf $model_columns_file

# 数据转换
model_convert_file=${SERVER_REPOSITORIES_PATH}/models/${name}/data.go
if [ ! -f "${model_convert_file}" ]; then
    touch $model_convert_file
fi
# 数据库操作
model_db_file=${SERVER_REPOSITORIES_PATH}/models/${name}/db.go
if [ ! -f "${model_db_file}" ]; then
    touch $model_db_file
fi
# 创建目录
mkdir -p ${SERVER_REPOSITORIES_PATH}/models/${name}
# 生成model
$sql2gorm -f $file_path -o ${model_type_file} -json -no-null -null-style ptr -pkg "${name}"
exitStateBack "sql2gorm 构建失败"

# model name
model_name=$(capitalize $name)

if [ "$(uname)" == "Darwin" ]; then
    sed -i "" '1d' $model_type_file
else
    sed -i '1d' $model_type_file
fi


# 包信息
content="package ${name}\n"
# 导入信息
content="${content}import (\n	\"${MODULE_NAME}\/core\/pkg\/functions\"\n	\"${MODULE_NAME}\/core\/pkg\/orm\"\n)\n"
content="${content}\nvar _ orm.Model = (*${model_name})(nil)"

if [ "$(uname)" == "Darwin" ]; then
    sed -i "" "s/package ${name}/${content}/g" $model_type_file
else
    sed -i "s/package ${name}/${content}/g" $model_type_file
fi
# 公共方法
#sed -i "" 's/}/\n	*orm.BaseModel\n}/g' $model_type_file
# model指针名称
model_pointer_name=$(toPascalCase $name)
model_pointer_name=$(lowercase $model_pointer_name)
# 结构体转map
method_content="func (${model_pointer_name} ${model_name}) ToMap() map[string]interface{} {\n	return functions.StructToMap(${model_pointer_name}, \"json\", nil)\n}"
# 所有字段
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) Columns() []string {\n    return Columns\n}"

# 数据转换
if [ -z "$(cat $model_convert_file)" ]; then
    cat>$model_convert_file<<EOF
package ${name}

import (
	"errors"

	"github.com/mitchellh/mapstructure"

	 "skeyevss/core/pkg/functions"
)

type Item struct {
	*${model_name}

    UseDBCache bool \`json:"-"\`
	// TODO 转换字段
}

func NewItem() *Item {
	return new(Item)
}

func (i *Item) ConvToModel(call func(*Item) *Item) (*${model_name}, error) {
    // TODO 数据转换
    if i.${model_name} == nil {
		return nil, nil
	}

	if call != nil {
		i = call(i)
	}

	return i.${model_name}, nil
}

func (i *Item) MapToModel(input map[string]interface{}) (*Item, error) {
	if input == nil {
		return nil, errors.New("input object is nil")
	}

	var model ${model_name}
	decoder, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			DecodeHook: mapstructure.DecodeHookFunc(functions.MapStructureHook),
			Result:     &model,
			// TagName:    "mapstructure",
		},
	)
	if err != nil {
		return nil, err
	}

	if err := decoder.Decode(input); err != nil {
		return nil, err
	}

	return &Item{${model_name}: &model}, nil
}

func (*Item) CheckMap(input map[string]interface{}) (map[string]interface{}, error) {
    if input == nil {
		return nil, errors.New("input is nil")
	}

	for column := range input {
		if !functions.Contains(column, Columns) {
			return nil, errors.New("column: " + column + " does not exist")
		}
	}

	return input, nil
}

EOF
fi

touch $model_columns_file
# 主键
const_primary_keys="const ("
primary_keys=""
# 字段信息
column_array="\nvar Columns = []string{"
columns="package ${name}\n"
columns="${columns}\nvar ("
while IFS= read -r line; do
    if [[ "${line}" =~ column: ]]; then
        # 字段值
        column_val=$(echo "${line}" | awk -F'column:' '{split($2, a, /(;|")/); if (length(a[1]) > 0) print a[1]; else print "No semicolon found"}')
        # 字段名首字母大写
        column_name=$(capitalize $column_val)
        # 获取主键
        if echo "${line}" | grep -qE "primary_key|index:.*unique"; then
            const_primary_keys="${const_primary_keys}\n   Primary${column_name} = \"${column_val}\""

            if [ -z "$variable" ]; then
                primary_keys="${primary_keys}Primary${column_name}"
            else
                primary_keys="${primary_keys}, Primary${column_name}"
            fi
        fi
        columns="${columns}\n   Column${column_name} = \"${column_val}\""
        column_array="${column_array}\n   Column${column_name},"
    fi
done <$model_type_file

if [ -z "$primary_keys" ]; then
    # 主键
    const_primary_keys="const ("
    primary_keys=""
    # 字段信息
    column_array="\nvar Columns = []string{"
    columns="package ${name}\n"
    columns="${columns}\nvar ("
    while IFS= read -r line; do
        if [[ "${line}" =~ column: ]]; then
            # 字段值
            column_val=$(echo "${line}" | awk -F'column:' '{split($2, a, /(;|")/); if (length(a[1]) > 0) print a[1]; else print "No semicolon found"}')
            # 字段名首字母大写
            column_name=$(capitalize $column_val)
            # 获取主键
            if echo "${line}" | grep -qE "gorm:\"column:uniqueId;NOT NULL\" json:\"uniqueId\""; then
                const_primary_keys="${const_primary_keys}\n   Primary${column_name} = \"${column_val}\""

                if [ -z "$variable" ]; then
                    primary_keys="${primary_keys}Primary${column_name}"
                else
                    primary_keys="${primary_keys}, Primary${column_name}"
                fi
            fi
            columns="${columns}\n   Column${column_name} = \"${column_val}\""
            column_array="${column_array}\n   Column${column_name},"
        fi
    done <$model_type_file
fi

columns="${columns}\n)\n"
column_array="${column_array}\n}\n"
const_primary_keys="${const_primary_keys}\n)"
column_array="${column_array}\n${const_primary_keys}"
echo -e "${columns}${column_array}" > $model_columns_file

#sed -i "" "s/}/\n    	*orm.DefaultModel\n}/g" $model_type_file
#sed -i "" "$(grep -n "}" $model_type_file |awk -F":" '{print $1}' |head -n 1) i\\\n    *orm.DefaultModel\n" $model_type_file
#sed -i '' '0,/\}/s/\}/aaa/ ' $model_type_file
awk '{
    if (!replaced && match($0, /}/)) {
        sub(/}/, "\n    	*orm.DefaultModel\n}");
        replaced = 1;
    }
    print;
}' $model_type_file > temp.txt && \mv temp.txt $model_type_file

awk '{if(!found && /func \(m \*/) {found=1; next} if(found && /}/) {found=0; next} if(!found) print}' $model_type_file > temp.txt && \mv temp.txt $model_type_file

# 唯一索引
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) UniqueKeys() []string {\n	return []string{\n${primary_keys},\n}\n}"
# 主键
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) PrimaryKey() string {\n    return ${primary_keys}\n}"
# 表名
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) TableName() string {\n    return \"${name}\"\n}"
# 查询条件
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) QueryConditions(conditions []*orm.ConditionItem) []*orm.ConditionItem {\n    return conditions\n}"
# 设置更新条件
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) SetConditions(conditions []*orm.ConditionItem) []*orm.ConditionItem {\n    return conditions\n}"
# upsert 冲突字段
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) OnConflictColumns(_ []string) []string {\n    return nil\n}"
# 数据修正
method_content="${method_content}\n\n//  Correction 数据修正\nfunc (${model_pointer_name} ${model_name})  Correction(action orm.ActionType) interface{} {\n    if action == orm.ActionInsert {\n        ${model_pointer_name}.CreatedAt = uint64(functions.NewTimer().NowMilli())\n    }\n    ${model_pointer_name}.UpdatedAt = uint64(functions.NewTimer().NowMilli())\n\n    return ${model_pointer_name}\n}"
# map 数据修正
method_content="${method_content}\n\n// CorrectionMap map数据修正\nfunc (${model_pointer_name} ${model_name}) CorrectionMap(data map[string]interface{}) map[string]interface{} {\n	data[ColumnUpdatedAt] = uint64(functions.NewTimer().NowMilli())\n	return data\n}\n"
# 数据库缓存
method_content="${method_content}\n\n// UseCache 数据库缓存\nfunc (${model_pointer_name} ${model_name}) UseCache() *orm.UseCacheAdvanced {\n// TODO 数据库缓存\n//return &orm.UseCacheAdvanced{\n//	Query:   true,\n//	Update: true,\n//  CacheKeyPrefix: ${model_pointer_name}.TableName(),\n//	Driver: new(orm.CacheRedisDriver),\n//	Expire: 60,\n//}\nreturn nil\n}"
# 数据转换
method_content="${method_content}\n\n// ConvToItem 数据转换\nfunc (${model_pointer_name} ${model_name}) ConvToItem() (*Item, error) {\n// TODO 数据转换\nvar useDBCache = false\nif ${model_pointer_name}.DefaultModel != nil {\nuseDBCache = ${model_pointer_name}.DefaultModel.UseDBCache\n}\n\nreturn &Item{\n${model_name}:  &${model_pointer_name},\nUseDBCache: useDBCache,\n}, nil\n}"
method_content="${method_content}\n\nfunc (${model_pointer_name} ${model_name}) Conv(data interface{}) error {\n	b, err := functions.JSONMarshal(${model_pointer_name})\n	if err != nil {\n		return err\n	}\n\n	return functions.JSONUnmarshal(b, data)\n}"
# 方法实现
echo -e "${method_content}" >> $model_type_file

# 数据库操作
if [ -z "$(cat $model_db_file)" ]; then
    cat>$model_db_file<<EOF
package ${name}

import (
    "time"

    "gorm.io/gorm"

	 "skeyevss/core/pkg/orm"
)

type DB struct {
	*orm.Foundation[${model_name}]
}

func NewDB(db *gorm.DB) *DB {
	return &DB{orm.NewFoundation[${model_name}](db, ${model_name}{}, 5*time.Second)}
}
EOF
fi

# 格式化文件
$FORMATTER -rm-unused -set-alias -format $model_type_file
$FORMATTER -rm-unused -set-alias -format $model_columns_file
$FORMATTER -rm-unused -set-alias -format $model_convert_file
$FORMATTER -rm-unused -set-alias -format $model_db_file

# 还原sql
\mv $file_path_tmp $file_path
#tmp_file="temp.txt"
#grep "gorm:" $model_type_file > $tmp_file
#while read line; do
#    column=$(echo $line | awk -F"column:" '{print $2}' | awk -F";|\"" '{print $1}')
#    old=$(echo $line | awk -F"\`" '{print $1}')
#    replace=$(echo $line | awk -F"\`" '{print $2}')
#    sed -i "" "s#$line#${old}\`json:\"${column}\" ${replace}\`#g" $model_type_file
#done < $tmp_file
#
#rm -rf $tmp_file

