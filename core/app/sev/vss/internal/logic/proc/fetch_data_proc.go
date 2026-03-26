package proc

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"

	"skeyevss/core/app/sev/db/client/configservice"
	"skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/app/sev/vss/internal/pkg/rule"
	"skeyevss/core/app/sev/vss/internal/types"
	cTypes "skeyevss/core/common/types"
	"skeyevss/core/pkg/categories"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/pkg/xmap"
	"skeyevss/core/repositories/models/dictionaries"
	mediaServers "skeyevss/core/repositories/models/media-servers"
	"skeyevss/core/repositories/models/settings"
)

var _ types.SipProcLogic = (*FetchDataLogic)(nil)

var (
	fetch_1 sync.Once
	fetch_2 sync.Once
)

const wsDiscoveryMessage = `<?xml version="1.0" encoding="UTF-8"?>
<e:Envelope xmlns:e="http://www.w3.org/2003/05/soap-envelope"
            xmlns:w="http://schemas.xmlsoap.org/ws/2004/08/addressing"
            xmlns:d="http://schemas.xmlsoap.org/ws/2005/04/discovery"
            xmlns:dn="http://www.onvif.org/ver10/network/wsdl">
  <e:Header>
    <w:MessageID>uuid:84ede3de-7dec-11d0-c360-f01234567890</w:MessageID>
    <w:To e:mustUnderstand="true">urn:schemas-xmlsoap-org:ws:2005:04:discovery</w:To>
    <w:Action a:mustUnderstand="true">http://schemas.xmlsoap.org/ws/2005/04/discovery/Probe</w:Action>
  </e:Header>
  <e:Body>
    <d:Probe>
      <d:Types>dn:NetworkVideoTransmitter</d:Types>
    </d:Probe>
  </e:Body>
</e:Envelope>`

type FetchDataLogic struct {
	svcCtx      *types.ServiceContext
	recoverCall func(name string)
}

func (l *FetchDataLogic) DO(params *types.DOProcLogicParams) {
	l = &FetchDataLogic{
		svcCtx:      params.SvcCtx,
		recoverCall: params.RecoverCall,
	}
	defer l.recoverCall("获取数据")

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		defer func() {
			fetch_1.Do(func() {
				l.svcCtx.InitFetchDataState.Done()
			})
		}()
		l.dictionaries()
	}()

	go func() {
		defer wg.Done()
		defer func() {
			fetch_2.Do(func() {
				l.svcCtx.InitFetchDataState.Done()
			})
		}()
		l.setting()
		// 依赖设置获取
		l.mediaServers()
		// 设备发现
		l.onvifDiscover()
		// 获取所有通道和设备在线状态
		l.deviceOnlineState()
	}()

	wg.Wait()

	for v := range time.NewTicker(time.Second).C {
		var now = v.Unix()
		if now%4 == 0 {
			go l.dictionaries()
			go l.setting()
			go l.mediaServers()
		}

		if now%120 == 0 {
			go l.onvifDiscover()
		}

		if now%10 == 0 {
			go l.deviceOnlineState()
		}
	}
}

// 字典获取
func (l *FetchDataLogic) dictionaries() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := response.NewRpcToHttpResp[*configservice.Response, []*categories.Item[int, *dictionaries.Item]]().Parse(
		func() (*configservice.Response, error) {
			return l.svcCtx.RpcClients.Config.DictionaryTrees(ctx, &configservice.EmptyRequest{})
		},
	)
	if err != nil {
		functions.LogError("字典trees获取失败, err: ", err.Error)
		return
	}

	var maps = make(map[string]*categories.Item[int, *dictionaries.Item])
	for _, item := range res.Data {
		maps[item.Raw.UniqueId] = item
	}

	l.svcCtx.DictionaryMap = maps
}

// 设置获取
func (l *FetchDataLogic) setting() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := response.NewRpcToHttpResp[*configservice.Response, *settings.Item]().Parse(
		func() (*configservice.Response, error) {
			return l.svcCtx.RpcClients.Config.SettingRow(ctx, &configservice.EmptyRequest{})
		},
	)
	if err != nil {
		functions.LogError("设置获取失败, err: ", err.Error)
		return
	}

	l.svcCtx.Setting = rule.NewConfig(l.svcCtx.Config, res.Data).Conv().Setting()
}

// media server records
func (l *FetchDataLogic) mediaServers() {
	type listResp struct {
		List  []*mediaServers.Item `json:"list,omitempty,optional"`
		Count int64                `json:"count,omitempty,optional"`
	}

	// 获取资源
	res, err := response.NewRpcToHttpResp[*configservice.Response, *listResp]().Parse(
		func() (*configservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(&orm.ReqParams{All: true})
			if err != nil {
				return nil, err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return l.svcCtx.RpcClients.Config.MsList(ctx, data)
		},
	)
	if err != nil {
		functions.LogError("media server res 获取失败, err", err.Error)
		return
	}

	l.svcCtx.MediaServerRecords = res.Data.List
}

// 设备探测
var onvifKeyMaps = xmap.New[string, string](100)

func (l *FetchDataLogic) onvifDiscover() {
	var discoveryTimeout = time.Duration(l.svcCtx.Config.Onvif.DiscoveryTimeout) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), discoveryTimeout)
	defer cancel()

	// 设置UDP连接
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		functions.LogError(fmt.Errorf("onvif discover failed to create UDP connection: %v", err))
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	data := []byte(wsDiscoveryMessage)
	p := ipv4.NewPacketConn(conn)
	// 获取多播地址
	dst, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", l.svcCtx.Config.Onvif.MulticastIP, l.svcCtx.Config.Onvif.WsDiscoveryPort))
	if err != nil {
		functions.LogError(fmt.Errorf("onvif discover failed to resolve multicast address: %v", err))
		return
	}

	// 获取所有网卡进行组播发Onvif发现包（合理吧）
	iFaces, err := net.Interfaces()
	if err != nil {
		functions.LogError("Failed to get net interfaces: %v", err)
		return
	}
	for _, ifi := range iFaces {
		if err := p.JoinGroup(&ifi, dst); err != nil {
			continue
		}
		if err := p.SetMulticastInterface(&ifi); err != nil {
			continue
		}
		_ = p.SetMulticastTTL(2)
		if _, err := p.WriteTo(data, nil, dst); err != nil {
			continue
		}
	}
	// 离开组播
	defer func() {
		_ = p.LeaveGroup(nil, dst)
		_ = p.Close()
	}()

	// 设置UDP发送超时
	if err := p.SetDeadline(time.Now().Add(discoveryTimeout)); err != nil {
		functions.LogError(fmt.Errorf("onvif discover Failed to set deadline: %v", err))
		return
	}

	// 收集发现的设备
	var (
		devices = xmap.New[string, *cTypes.OnvifDeviceItem](100)
		timeout = time.After(discoveryTimeout)
		buffer  = make([]byte, 10240)
		done    = func() {
			var (
				records = make([]*cTypes.OnvifDeviceItem, devices.Len())
				i       = 0
			)
			for _, item := range devices.Values() {
				records[i] = item
				i += 1
			}

			l.svcCtx.OnvifDiscoverDevices = records
		}
	)

	for {
		select {
		case <-timeout: // 超时，返回已发现的设备
			done()
			return

		case <-ctx.Done(): // 上下文取消，返回已发现的设备
			done()
			return

		default:
			// 非阻塞读取响应
			n, addr, err := conn.ReadFrom(buffer)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					continue
				}

				functions.LogError("Failed to read response, err: ", err)
				continue
			}

			// 解析SOAP响应
			var resp types.OnvifWSDiscoveryResponse
			if err := xml.Unmarshal(buffer[:n], &resp); err != nil {
				functions.LogError(fmt.Sprintf("Failed to parse response from %s: %v", addr.String(), err))
				continue
			}

			for _, value := range resp.Body.ProbeMatches.Matches {
				uniqueId, ok := onvifKeyMaps.Get(value.EndpointReference.Address)
				if !ok {
					uniqueId = functions.GenerateUniqueID(8)
					onvifKeyMaps.Set(value.EndpointReference.Address, uniqueId)
				}

				var (
					uuid = strings.TrimPrefix(value.EndpointReference.Address, "urn:uuid:")
					// 获取XAddrs (可能包含多个地址)
					addresses = strings.Split(value.XAddrs, " ")
				)
				if len(addresses) <= 0 || addresses[0] == "" {
					continue
				}

				var (
					// 获取设备类型
					Types = strings.Split(value.Types, " ")
					// 获取设备范围
					scopes = strings.Split(value.Scopes, " ")
					// 提取设备名称和其他信息
					name, model, manufacturer string
				)
				for _, item := range scopes {
					var parts = strings.Split(item, "/")
					if strings.Contains(item, "name") {
						if len(parts) > 0 {
							name = parts[len(parts)-1]
						}
						continue
					}

					if strings.Contains(item, "hardware") {
						if len(parts) > 0 {
							model = parts[len(parts)-1]
						}
						continue
					}

					if strings.Contains(item, "manufacturer") {
						if len(parts) > 0 {
							manufacturer = parts[len(parts)-1]
						}
					}
				}

				devices.Set(uniqueId, &cTypes.OnvifDeviceItem{
					UUID:         uniqueId,
					OriginalUid:  uuid,
					Name:         name,
					Address:      strings.Split(addresses[0], "/")[2], // 提取IP地址
					ServiceURLs:  addresses,
					Types:        Types,
					XAddrs:       addresses,
					Scopes:       scopes,
					Model:        model,
					Manufacturer: manufacturer,
				})
			}
		}
	}
}

func (l *FetchDataLogic) deviceOnlineState() {
	res, err := response.NewRpcToHttpResp[*deviceservice.Response, *cTypes.DeviceOnlineStateResp]().Parse(
		func() (*deviceservice.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(new(orm.ReqParams))
			if err != nil {
				return nil, err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			return l.svcCtx.RpcClients.Device.OnlineState(ctx, data)
		},
	)
	if err != nil {
		functions.LogError("设备在线状态获取失败, err:", err)
		return
	}

	if res.Data != nil {
		l.svcCtx.DeviceOnlineState = res.Data
	}
}
