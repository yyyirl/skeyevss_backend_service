package {{.PkgModuleName}}

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/structpb"

	"skeyevss/core/app/sev/backend/internal/svc"
	"skeyevss/core/app/sev/backend/internal/types"
	"skeyevss/core/app/sev/db/client/{{.ServiceName}}"
	"skeyevss/core/common/opt"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/functions"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/system-operation-logs"
)

type CreateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLogic {
	return &CreateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateLogic) Create(req *types.RecordReq) (interface{}, *response.HttpErr) {
	// 日志记录
	opt.NewSystemOperationLogs(l.svcCtx.RpcClients).Make(l.ctx, systemOperationLogs.Types[systemOperationLogs.Type{{.LogType}}Create], req)

	res, err := response.NewRpcToHttpResp[*{{.ServiceName}}.Response, uint64]().Parse(
		func() (*{{.ServiceName}}.Response, error) {
			data, err := structpb.NewStruct(req.Record)
			if err != nil {
				return nil, err
			}

			return l.svcCtx.RpcClients.{{.ServiceClient}}.{{.ServiceModuleNameSingular}}Create(l.ctx, &{{.ServiceName}}.MapReq{Data: data})
		},
	)

	if err != nil {
		return 0, err
	}

	id, err1 := functions.ConvBytes[uint64](res.Res.Data)
	if err1 != nil {
		return 0, response.MakeError(response.NewHttpRespMessage().Err(err1), localization.MR1002)
	}
	return id, nil
}
