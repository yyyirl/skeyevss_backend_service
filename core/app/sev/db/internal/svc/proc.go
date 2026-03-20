// @Title        proc
// @Description  main
// @Create       yiyiyi 2025/12/2 17:42

package svc

import (
	"strconv"
	"time"

	"skeyevss/core/pkg/dt"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/repositories/models/channels"
	"skeyevss/core/repositories/models/devices"
)

func proc() {
	go deviceDepIdSet()
}

func deviceDepIdSet() {
	for {
		select {
		case <-svcCtx.DeviceDepIdSetChan:
			dt.TrailingDebounce("deviceDepIdSet", 2*time.Second, func() {
				doSetDeviceDepId()
			})
		}
	}
}

func doSetDeviceDepId() {
	functions.LogInfo(">>>>>>>>>>>>>> [START] 设置设备与组织部门关联关系")

	// 获取所有通道
	channelList, err := svcCtx.ChannelsModel.List(&orm.ReqParams{All: true})
	if err != nil {
		functions.LogError("设置设备与组织部门关联关系 通道获取失败, err: ", err)
		return
	}

	if len(channelList) <= 0 {
		functions.LogInfo(">>>>>>>>>>>>>> [STOP] 通道列表为空 忽略更新")
		return
	}

	var maps = make(map[string][]uint64)
	for _, item := range channelList {
		v, err := item.ConvToItem()
		if err != nil {
			functions.LogError("设置设备与组织部门关联关系 通道记录转换失败, err: ", err)
			return
		}

		if len(maps[v.DeviceUniqueId]) <= 0 {
			maps[v.DeviceUniqueId] = make([]uint64, 0)
		}
		maps[v.DeviceUniqueId] = append(maps[v.DeviceUniqueId], v.DepIds...)
	}

	for key, item := range maps {
		maps[key] = functions.ArrUnique(item)
	}

	// 重置所有设备记录
	if err := svcCtx.DevicesModel.UpdateWithParams(
		map[string]interface{}{
			devices.ColumnDepIds:    "[]",
			devices.ColumnUpdatedAt: functions.NewTimer().NowMilli(),
		},
		&orm.ReqParams{
			Conditions: []*orm.ConditionItem{
				{Column: devices.ColumnID, Value: 0, Operator: ">"},
			},
		},
	); err != nil {
		functions.LogError("设置设备与组织部门关联关系 初始化设备depIds失败 err: ", err)
		return
	}

	// 设置设备 depIds
	var records []*orm.BulkUpdateInner
	for key, item := range maps {
		var val = item
		if len(val) <= 0 {
			val = []uint64{}
		}

		records = append(records, &orm.BulkUpdateInner{
			PK:  key,
			Val: val,
		})
	}

	if len(records) > 0 {
		var (
			page  = 0
			limit = 50
		)
		functions.PickOffsetRangeWithCall(page, limit, len(records), func(start, end int) {
			if err := svcCtx.DevicesModel.BulkUpdate(
				devices.ColumnDeviceUniqueId,
				[]string{devices.ColumnUpdatedAt, devices.ColumnDepIds},
				[]*orm.BulkUpdateItem{
					{
						Column:  channels.ColumnDepIds,
						Records: records[start:end],
						Def:     &orm.BulkUpdateItemDef{Value: "[]", Type: 1},
					},
				},
			); err != nil {
				functions.LogError("设置设备与组织部门关联关系 数据批量更新失败, record["+strconv.Itoa(start)+":"+strconv.Itoa(end)+"] err: ", err)
			}
		})
	}

	functions.LogInfo(">>>>>>>>>>>>>> [SUCCESS]设置设备与组织部门关联关系")
}
