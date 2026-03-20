package svc

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"skeyevss/core/app/sev/backend/internal/config"
	"skeyevss/core/app/sev/backend/internal/middleware"
	backendClient "skeyevss/core/app/sev/db/client/backendservice"
	configClient "skeyevss/core/app/sev/db/client/configservice"
	deviceClient "skeyevss/core/app/sev/db/client/deviceservice"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/common/client"
	"skeyevss/core/common/types"
	cTypes "skeyevss/core/common/types"
	"skeyevss/core/pkg/contextx"
	"skeyevss/core/pkg/device"
	"skeyevss/core/pkg/dt"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/interceptor"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/dictionaries"
	mediaServers "skeyevss/core/repositories/models/media-servers"
	"skeyevss/core/repositories/models/settings"
)

type (
	Health struct {
		Hardware []*device.Hardware
		Services [][]*device.Sev
		MemTotal uint64
		Count    int
	}

	ServiceContext struct {
		Config config.Config
		BaseMiddleware,
		AuthMiddleware rest.Middleware

		// RedisClient *redis.Client

		RpcClients *client.GRPCClients
		Health     *Health

		MSSet,
		SettingSet,
		AuthSet,
		DictSet chan struct{}

		StartTimestamp int64
		BuildTime      string
	}
)

var (
	healthCache      = new(Health)
	settingRow       *settings.Item
	deviceStatistics *cTypes.DeviceStatisticsResp
	authRes          = new(types.AuthRes)
	dictRes          []*dictionaries.Item
	msRes            []*mediaServers.Item
)

func init() {
	healthCache.Hardware = []*device.Hardware{}
	healthCache.Services = [][]*device.Sev{}
	healthCache.Count = 30
}

func NewServiceContext(c config.Config, buildTime string) *ServiceContext {
	var (
		rpcInterceptor   = client.NewRpcClientInterceptor(c.RpcInterceptor)
		rpcClientOptions = rpcInterceptor.Options(map[client.OptionsKey]zrpc.ClientOption{
			client.OptionsRetryKey: zrpc.WithUnaryClientInterceptor(
				rpcInterceptor.RetryInterceptor(
					c.RpcInterceptor.RpcCallerRetryMax,
					time.Duration(c.RpcInterceptor.RpcCallerRetryWaitInterval)*time.Millisecond,
				),
			),
			client.OptionsKeepaliveKey: zrpc.WithDialOption(
				grpc.WithKeepaliveParams(keepalive.ClientParameters{
					Time:                time.Duration(c.RpcInterceptor.RpcKeepaliveTime) * time.Second,
					Timeout:             time.Duration(c.RpcInterceptor.RpcKeepaliveTimeout) * time.Second,
					PermitWithoutStream: c.RpcInterceptor.RpcKeepalivePermitWithoutStream,
				}),
			),
			client.OptionsApi2RpcKey: zrpc.WithUnaryClientInterceptor(
				rpcInterceptor.Api2DBRpc(
					&interceptor.RPCAuthSenderType{
						SKey: c.SevBase.Keys.DB,
						CKey: c.SevBase.Keys.BackendApi,
					},
				),
			),
		})
		svcCtx = &ServiceContext{
			Config: c,
			BaseMiddleware: func(next http.HandlerFunc) http.HandlerFunc {
				return middleware.NewBaseMiddleware().Handle(c, next, buildTime)
			},
			AuthMiddleware: func(next http.HandlerFunc) http.HandlerFunc {
				return func(writer http.ResponseWriter, request *http.Request) {
					middleware.NewAuthMiddleware().Handle(c, next, authRes, buildTime)(writer, request)
				}
			},

			// RedisClient: redis.New(c.Mode, c.Redis, c.Log),
			RpcClients: &client.GRPCClients{
				Backend: backendClient.NewBackendService(zrpc.MustNewClient(c.DBGrpc, rpcClientOptions...)),
				Config:  configClient.NewConfigService(zrpc.MustNewClient(c.DBGrpc, rpcClientOptions...)),
				Device:  deviceClient.NewDeviceService(zrpc.MustNewClient(c.DBGrpc, rpcClientOptions...)),
			},
			Health:         healthCache,
			SettingSet:     make(chan struct{}, 20),
			MSSet:          make(chan struct{}, 20),
			AuthSet:        make(chan struct{}, 20),
			DictSet:        make(chan struct{}, 20),
			StartTimestamp: functions.NewTimer().NowMilli(),
		}
	)

	// 首次启动检查rpc链接
	svcCtx.fetchSetting(true)

	go svcCtx.health(c)

	go svcCtx.auth()
	go svcCtx.fetchAuth()

	go svcCtx.dict()
	go svcCtx.fetchDict()

	go svcCtx.setting()
	go svcCtx.fetchSetting(false)

	go svcCtx.ms()
	go svcCtx.fetchMS()

	go svcCtx.fetchDeviceStatistics()
	go svcCtx.deviceStatistics()

	return svcCtx
}

// 服务健康检查
func (s *ServiceContext) health(conf config.Config) {
	<-time.After(time.Millisecond * 100)

	healthCache.MemTotal = device.NewSystem().MemTotal()
	for {
		// functions.LogInfo("health interval")
		healthCache.Hardware = append(healthCache.Hardware, device.NewSystem().Hardware(time.Second))
		healthCache.Services = append(
			healthCache.Services,
			device.NewSystem().Services(
				time.Second, []*device.SevConf{
					{
						Name: conf.SevBase.SevNameMysql,
						Port: conf.SevBase.MysqlPort,
					},
					{
						Name: conf.SevBase.SevNameRedis,
						Port: conf.SevBase.RedisPort,
					},
					{
						Name: conf.SevBase.SevNameEtcd,
						Port: conf.SevBase.EtcdPort,
					},
					{
						Name: conf.SevBase.SevNameMediaServer,
						Port: conf.SevBase.MediaServerPort,
					},
					{
						Name: conf.SevBase.SevNameVss,
						Port: conf.SevBase.VssPort,
					},
					{
						Name: conf.SevBase.SevNameCron,
						Port: conf.SevBase.CronPort,
					},
					{
						Name: conf.SevBase.SevNameDB,
						Port: conf.SevBase.DBPort,
					},
					{
						Name: conf.SevBase.SevNameBackendApi,
						Port: conf.SevBase.BackendApiPort,
					},
					{
						Name: conf.SevBase.SevNameWebSev,
						Port: conf.SevBase.WebSevPort,
					},
				},
			),
		)
		if len(healthCache.Hardware) >= healthCache.Count {
			healthCache.Hardware = healthCache.Hardware[len(healthCache.Hardware)-healthCache.Count:]
		}

		if len(healthCache.Services) >= healthCache.Count {
			healthCache.Services = healthCache.Services[len(healthCache.Services)-healthCache.Count:]
		}
	}
}

// ----------------------------------------------- 权限

func (s *ServiceContext) auth() {
	for {
		select {
		case <-s.AuthSet:
			dt.TrailingDebounce(
				"fetchAuthRes",
				500*time.Millisecond,
				func() {
					s.fetchAuth()
				},
			)
		}
	}
}

func (s *ServiceContext) fetchAuth() {
	// 获取资源
	res, err := response.NewRpcToHttpResp[*backendClient.Response, *types.AuthRes]().Parse(
		func() (*backendClient.Response, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			return s.RpcClients.Backend.AuthRes(ctx, &backendClient.EmptyRequest{})
		},
	)
	if err != nil {
		functions.LogError("auth res 获取失败, err", err.Error)
		return
	}

	authRes = res.Data
}

// ----------------------------------------------- 权限

// ----------------------------------------------- 字典

func (s *ServiceContext) dict() {
	for {
		select {
		case <-s.DictSet:
			dt.TrailingDebounce(
				"fetchDictRes",
				2*time.Second,
				func() {
					s.fetchDict()
				},
			)
		}
	}
}

func (s *ServiceContext) fetchDict() {
	type listResp struct {
		List  []*dictionaries.Item `json:"list,omitempty,optional"`
		Count int64                `json:"count,omitempty,optional"`
	}

	// 获取资源
	res, err := response.NewRpcToHttpResp[*configClient.Response, *listResp]().Parse(
		func() (*configClient.Response, error) {
			data, err := conv.New(s.Config.Mode).ToPBParams(&orm.ReqParams{All: true})
			if err != nil {
				return nil, err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return s.RpcClients.Config.Dictionaries(ctx, data)
		},
	)
	if err != nil {
		functions.LogError("字典 获取失败, err", err.Error)
		return
	}

	dictRes = res.Data.List
}

// 获取字典

func (s *ServiceContext) Dictionaries() []*dictionaries.Item {
	return dictRes
}

// ----------------------------------------------- 字典

// ----------------------------------------------- setting

func (s *ServiceContext) setting() {
	for {
		select {
		case <-s.SettingSet:
			dt.TrailingDebounce(
				"fetchSettingRes",
				2*time.Second,
				func() {
					s.fetchSetting(false)
				},
			)
		}
	}
}

// 获取系统设置
func (s *ServiceContext) fetchSetting(withExit bool) {
	// 获取资源
	res, err := response.NewRpcToHttpResp[*configClient.Response, *settings.Item]().Parse(
		func() (*configClient.Response, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return s.RpcClients.Config.SettingRow(ctx, &configClient.EmptyRequest{})
		},
	)
	if err != nil {
		if withExit {
			panic(fmt.Errorf("setting res 获取失败, err: %s", err.Error))
		} else {
			functions.LogError("setting res 获取失败, err", err.Error)
		}

		return
	}

	if res != nil {
		settingRow = res.Data
	}
}

// 获取setting

func (s *ServiceContext) Settings() *settings.Item {
	settingRow.ItemCorrection(&settings.ItemCorrectionParams{
		BaseConf:   s.Config.SevBase,
		SipConf:    s.Config.Sip,
		InternalIp: s.Config.InternalIP,
		ExternalIp: s.Config.ExternalIP,
	})

	return settingRow
}

// ----------------------------------------------- setting

// ----------------------------------------------- media server records

func (s *ServiceContext) ms() {
	for {
		select {
		case <-s.MSSet:
			dt.TrailingDebounce(
				"fetchMediaServerRecords",
				2*time.Second,
				func() {
					s.fetchMS()
				},
			)
		}
	}
}

// 获取ms records

func (s *ServiceContext) fetchMS() {
	type listResp struct {
		List  []*mediaServers.Item `json:"list,omitempty,optional"`
		Count int64                `json:"count,omitempty,optional"`
	}

	// 获取资源
	res, err := response.NewRpcToHttpResp[*configClient.Response, *listResp]().Parse(
		func() (*configClient.Response, error) {
			data, err := conv.New(s.Config.Mode).ToPBParams(&orm.ReqParams{All: true})
			if err != nil {
				return nil, err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return s.RpcClients.Config.Dictionaries(ctx, data)
		},
	)
	if err != nil {
		functions.LogError("Media server 获取失败, err", err.Error)
		return
	}

	msRes = res.Data.List
}

// 获取ms

func (s *ServiceContext) MediaServerRecords() []*mediaServers.Item {
	return msRes
}

func (s *ServiceContext) MSVoteNode(ids []uint64) *cTypes.MSVoteNodeResp {
	// 默认节点
	var (
		mediaServerInternalIP   = s.Config.InternalIP
		mediaServerInternalPort = s.Config.SevBase.MediaServerPort

		mediaServerExternalIP   = s.Config.ExternalIP
		mediaServerExternalPort = s.Config.SevBase.MediaServerPort

		node string
	)
	if mediaServerInternalIP != "" && mediaServerInternalPort > 0 {
		node = fmt.Sprintf("%s:%d", mediaServerInternalIP, mediaServerInternalPort)
	} else if mediaServerExternalIP != "" && mediaServerExternalPort > 0 {
		node = fmt.Sprintf("%s:%d", mediaServerExternalIP, mediaServerExternalPort)
	}

	if len(ids) <= 0 {
		return &cTypes.MSVoteNodeResp{
			Address: node,
			Name:    "default",
		}
	}

	var nodes []*cTypes.MSVoteNodeResp
	for _, item := range s.MediaServerRecords() {
		if functions.Contains(item.ID, ids) && item.IP != "" && item.Port > 0 {
			nodes = append(nodes, &cTypes.MSVoteNodeResp{
				Address: fmt.Sprintf("%s:%d", item.IP, item.Port),
				Name:    item.Name,
				ID:      item.ID,
			})
		}
	}

	if len(nodes) <= 0 {
		return &cTypes.MSVoteNodeResp{
			Address: node,
			Name:    "default",
		}
	}

	if len(nodes) <= 1 {
		return &cTypes.MSVoteNodeResp{
			Address: node,
			Name:    "default",
		}
	}

	return nodes[rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(nodes))]
}

// ----------------------------------------------- media server records

// ----------------------------------------------- device statistics

func (s *ServiceContext) deviceStatistics() {
	var ticker = time.NewTicker(time.Second * 3)
	for {
		select {
		case <-ticker.C:
			s.fetchDeviceStatistics()
		}
	}
}

// 获取系统设置
func (s *ServiceContext) fetchDeviceStatistics() {
	res, err := response.NewRpcToHttpResp[*deviceClient.Response, *cTypes.DeviceStatisticsResp]().Parse(
		func() (*deviceClient.Response, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return s.RpcClients.Device.DeviceOnlineStatistics(ctx, &deviceClient.EmptyRequest{})
		},
	)
	if err != nil {
		functions.LogError("setting res 获取失败, err", err.Error)
		return
	}

	if res != nil {
		deviceStatistics = res.Data
	}
}

func (s *ServiceContext) DeviceStatistics() *cTypes.DeviceStatisticsResp {
	return deviceStatistics
}

// ----------------------------------------------- device statistics

func (s *ServiceContext) RemoteReq(ctx context.Context) *cTypes.RemoteReq {
	var (
		vssHttpTarget = s.Config.VssHttpTarget
		vssHttpUrl    = fmt.Sprintf("http://%s", s.Config.VssHttpTarget)
		vssSseTarget  = s.Config.VssSseTarget
		vssSseUrl     = fmt.Sprintf("http://%s", s.Config.VssSseTarget)
		requestInfo   = contextx.GetCtxRequestInfo(ctx)
		referer       = ""
	)
	if tmp, ok := requestInfo["referer"]; ok {
		referer, ok = tmp.(string)
		if ok {
			if parsedURL, _ := url.Parse(referer); parsedURL != nil {
				var (
					host       = strings.Split(parsedURL.Host, ":")[0]
					isInternal = functions.Contains(
						host,
						[]string{
							"127.0.0.1", "::1", "localhost",
						},
					) || host == s.Config.InternalIP
				)
				if !isInternal {
					if v := strings.Split(s.Config.VssHttpTarget, ":"); len(v) == 2 {
						vssHttpUrl = fmt.Sprintf("http://%s:%s", s.Config.ExternalIP, v[1])
						vssHttpTarget = fmt.Sprintf("%s:%s", s.Config.ExternalIP, v[1])
					}

					if v := strings.Split(s.Config.VssSseTarget, ":"); len(v) == 2 {
						vssSseUrl = fmt.Sprintf("http://%s:%s", s.Config.ExternalIP, v[1])
						vssSseTarget = fmt.Sprintf("%s:%s", s.Config.ExternalIP, v[1])
					}
				}
			}
		}
	}

	return &cTypes.RemoteReq{
		VssHttpTarget:      vssHttpTarget,
		VssHttpUrl:         vssHttpUrl,
		VssHttpUrlInternal: fmt.Sprintf("http://%s", s.Config.VssHttpTarget),
		VssSseTarget:       vssSseTarget,
		VssSseUrl:          vssSseUrl,
		Referer:            referer,
	}
}
