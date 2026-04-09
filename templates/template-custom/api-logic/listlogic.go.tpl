package {{.PkgModuleName}}

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/backend/internal/svc"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/app/sev/db/client/{{.ServiceName}}"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/{{.ModelName}}"
)

type ListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListLogic {
	return &ListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListLogic) List(req *orm.ReqParams) (interface{}, *response.HttpErr) {
	res, err := response.NewRpcToHttpResp[*{{.ServiceName}}.Response, *response.ListResp[[]*{{.ModuleName}}.Item]]().Parse(
		func() (*{{.ServiceName}}.Response, error) {
			data, err := conv.New(l.svcCtx.Config.Mode).ToPBParams(req)
            if err != nil {
                return nil, err
            }

			return l.svcCtx.RpcClients.{{.ServiceClient}}.{{.ServiceModuleNamePlural}}(l.ctx, data)
		},
	)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}
