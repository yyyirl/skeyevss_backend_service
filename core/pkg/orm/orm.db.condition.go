package orm

import (
	"errors"
	"fmt"
	"strings"

	"skeyevss/core/pkg/functions"
)

type (
	ConditionOriginalItem struct {
		Query  string
		Values []interface{}
	}

	ConditionItem struct {
		Column   string        `json:"column"`
		Value    interface{}   `json:"value,optional"`
		Values   []interface{} `json:"values,optional"`
		Operator string        `json:"operator,optional"`
		UseNil   bool          `json:"-"`

		// 原始条件
		Original *ConditionOriginalItem `json:"original,optional"`

		// 与兄弟条件关系
		LogicalOperator string           `json:"logicalOperator,optional"`
		Inner           []*ConditionItem `json:"inner,optional"`

		Columns []string `json:"-"`
	}

	ConditionBuild[T Model] struct {
		conditions    []*ConditionItem
		originalModel T
		databaseType  string
	}
)

func (c *ConditionBuild[T]) validateOriginalQuery(query string, values []interface{}) error {
	var q = strings.TrimSpace(query)
	if q == "" {
		return errors.New("original query 不能为空")
	}

	// 禁止明显的多语句/注释注入
	if strings.Contains(q, ";") || strings.Contains(q, "--") || strings.Contains(q, "/*") || strings.Contains(q, "*/") {
		return errors.New("original query 包含非法字符")
	}

	// 禁止 DDL/DML 关键字（防止把 where 片段变成完整语句）
	var lower = strings.ToLower(q)
	for _, kw := range []string{"select ", "insert ", "update ", "delete ", "drop ", "truncate ", "alter ", "create "} {
		if strings.Contains(lower, kw) {
			return errors.New("original query 包含非法关键字")
		}
	}

	// 校验占位符数量
	var (
		placeholderCount = strings.Count(q, "?")
		valueCount       = len(values)
	)
	if placeholderCount != valueCount {
		return fmt.Errorf("original query 占位符数量不匹配: 期望 %d, 实际 %d", placeholderCount, valueCount)
	}

	return nil
}

func (c *ConditionBuild[T]) quoteColumn(column string) string {
	switch strings.ToLower(c.databaseType) {
	case DBTypePostgres:
		return `"` + column + `"`

	case DBTypeSqlserver:
		return `[` + column + `]`

	case DBTypeMysql, DBTypeSqlite:
		fallthrough

	default:
		return "`" + column + "`"
	}
}

func (c *ConditionItem) operatorValidate() bool {
	if c.Operator == "" {
		c.Operator = "="
		return true
	}

	switch c.Operator {
	case "<", "<=", ">", ">=", "!=", "in", "IN", "notin", "NOTIN", "=", "like", "LIKE", "llike", "LLIKE", "jin", "JIN":
		return true
	}

	return false
}

func (c *ConditionItem) logicalOperatorValidate() bool {
	if c.LogicalOperator == "" {
		c.LogicalOperator = "AND"
		return true
	}

	for _, item := range LogicalOperators {
		if c.LogicalOperator == item {
			return true
		}
	}

	return false
}

func (c *ConditionItem) columnValidate() error {
	var columnSign = c.Column != ""
	if !columnSign && len(c.Inner) <= 0 {
		return errors.New("condition item column,inner 不能同时为空")
	}

	if columnSign {
		if c.Value == nil && len(c.Values) <= 0 {
			return fmt.Errorf("condition item [%s] value values 不能同时为空", c.Column)
		}

		if !functions.Contains(c.Column, c.Columns) {
			return fmt.Errorf("条件字段[%s]不存在", c.Column)
		}
	}

	return nil
}

func NewConditionBuild[T Model](conditions []*ConditionItem, model T, databaseType string) *ConditionBuild[T] {
	return &ConditionBuild[T]{
		conditions:    conditions,
		originalModel: model,
		databaseType:  databaseType,
	}
}

func (c *ConditionBuild[T]) Do(emptyCondition bool) (string, []interface{}, error) {
	if len(c.conditions) <= 0 {
		if emptyCondition {
			return "", nil, nil
		}

		return "", nil, errors.New("conditions is nil")
	}

	var (
		columns      = c.originalModel.Columns()
		wheres       []string
		placeholders []interface{}
	)
	for _, item := range c.conditions {
		item.Columns = columns
		if item.Original == nil {
			if err := item.columnValidate(); err != nil {
				return "", nil, err
			}
		}

		if !item.operatorValidate() {
			return "", nil, errors.New("condition item operator 值不匹配")
		}

	RETRY:
		if !item.logicalOperatorValidate() {
			return "", nil, errors.New("condition item logicalOperator 值不匹配")
		}

		if len(item.Inner) > 0 {
			whereStr, args, err := NewConditionBuild[T](item.Inner, c.originalModel, c.databaseType).Do(emptyCondition)
			if err != nil {
				return "", nil, err
			}

			whereStr = fmt.Sprintf(" %s %s", item.LogicalOperator, whereStr)
			for _, item := range LogicalOperators {
				whereStr = strings.Trim(whereStr, " "+item+" ")
			}

			wheres = append(wheres, fmt.Sprintf(" %s (%s)", item.LogicalOperator, whereStr))
			placeholders = append(placeholders, args...)
			continue
		}

		if item.Original != nil && item.Original.Query != "" {
			// if err := c.validateOriginalQuery(item.Original.Query, item.Original.Values); err != nil {
			// 	return "", nil, err
			// }

			wheres = append(wheres, fmt.Sprintf(" %s (%s)", item.LogicalOperator, item.Original.Query))
			placeholders = append(placeholders, item.Original.Values...)
			continue
		}

		if len(item.Values) > 0 {
			if strings.ToLower(item.Operator) == "jin" {
				item.Original = NewExternalDB(c.databaseType).MakeCaseJSONContainsCondition(item.Column, item.Values)
				goto RETRY
			}

			var operator = "in"
			if strings.ToLower(item.Operator) == "notin" {
				operator = "not in"
			}

			wheres = append(
				wheres,
				fmt.Sprintf(
					" %s %s %s (%s)",
					item.LogicalOperator,
					c.quoteColumn(item.Column),
					operator,
					strings.Trim(strings.Repeat("?,", len(item.Values)), ","),
				),
			)

			placeholders = append(placeholders, item.Values...)
			continue
		}

		var (
			op     = strings.ToLower(item.Operator)
			column = c.quoteColumn(item.Column)
		)
		wheres = append(wheres, fmt.Sprintf(" %s %s %s ?", item.LogicalOperator, column, item.Operator))
		if op == "like" {
			placeholders = append(placeholders, fmt.Sprintf("%%%v%%", item.Value))
			continue
		}

		if op == "llike" {
			placeholders = append(placeholders, fmt.Sprintf("%%%v", item.Value))
			continue
		}

		placeholders = append(placeholders, item.Value)
	}

	var where string
	for _, item := range wheres {
		where += item
	}

	for _, item := range LogicalOperators {
		where = strings.Trim(where, " "+item+" ")
	}

	return where, placeholders, nil
}
