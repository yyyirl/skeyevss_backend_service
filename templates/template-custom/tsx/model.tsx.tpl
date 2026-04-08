import { RenderStyle } from '#types/ant.table.d'
import { type RowDataType } from '#types/base.d'
import { type XFormItem, XFormItemType } from '#types/ant.form.d'
import { type RowType } from '#components/table/model'

export class Item implements RowDataType {
	{{.Model}}

	primaryKeyValue(): string | number {
		return this.id
	}

	primaryKeyColumn(): keyof Item {
		return 'id'
	}

	static primaryKeyColumn(): keyof Item {
		return 'id'
	}

	hiddenEdit(): boolean {
		return false
	}

	hiddenDelete(): boolean {
		return false
	}

	hiddenChecked(): boolean {
		return false
	}

	updateProperty(column: keyof this, value: any): Item {
		const data = { ...this }
		data[ column ] = value
		return new Item(data)
	}

	static conv(data: object): Item {
		return new Item(data)
	}
}

export const columns = (): RowType<Item> => [
	{
		title: 'id',
		dataIndex: 'id',
		sorter: true,
		width: 80,
		fixed: 'left'
	},
	{
		title: '更新时间',
		dataIndex: 'updatedAt',
		width: 200,
		sorter: true,
		renderStyle: RenderStyle.timestamp
	},
	{
		title: '创建时间',
		dataIndex: 'createdAt',
		width: 200,
		sorter: true,
		renderStyle: RenderStyle.timestamp
	}
]

export const formColumns = (): Array<XFormItem<Item>> => [
	{
		dataIndex: 'username',
		defValue: '',
		label: '标题',
		type: XFormItemType.input
	}
]