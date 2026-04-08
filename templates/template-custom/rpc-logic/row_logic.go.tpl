package {{.ServiceName}}logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/db/db"
	"skeyevss/core/app/sev/db/internal/svc"
	"skeyevss/core/pkg/response"
)

type {{.ServiceModuleNameSingular}}RowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.ServiceModuleNameSingular}}RowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.ServiceModuleNameSingular}}RowLogic {
	return &{{.ServiceModuleNameSingular}}RowLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.ServiceModuleNameSingular}}RowLogic) {{.ServiceModuleNameSingular}}Row(in *db.IDReq) (*db.Response, error) {
	row, err := l.svcCtx.{{.ServiceModuleNamePlural}}Model.Row(in.ID)
	if err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	data, err := row.ConvToItem()
	if err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	return response.NewRpcResp[*db.Response]().Make(data, 3, func(data []byte) *db.Response {
		return &db.Response{Data: data}
	})
}
