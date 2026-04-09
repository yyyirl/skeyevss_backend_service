package {{.PkgModuleName}}

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/backend/internal/svc"
	"skeyevss/core/app/sev/backend/internal/types"
	"skeyevss/core/app/sev/db/client/{{.ServiceName}}"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/{{.ModuleName}}"
)

type RowLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RowLogic {
	return &RowLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RowLogic) Row(req *types.IdQuery) (interface{}, *response.HttpErr) {
	res, err := response.NewRpcToHttpResp[*{{.ServiceName}}.Response, *{{.ModuleName}}.Item]().Parse(
		func() (*{{.ServiceName}}.Response, error) {
			return l.svcCtx.RpcClients.{{.ServiceClient}}.{{.ServiceModuleNameSingular}}Row(l.ctx, &{{.ServiceName}}.IDReq{ID: uint64(req.Id)})
		},
	)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}
