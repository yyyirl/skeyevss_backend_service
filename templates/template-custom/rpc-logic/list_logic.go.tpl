package {{.ServiceName}}logic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/db/db"
	"skeyevss/core/app/sev/db/internal/svc"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/{{.ModelName}}"
)

type {{.ServiceModuleNamePlural}}Logic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.ServiceModuleNamePlural}}Logic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.ServiceModuleNamePlural}}Logic {
	return &{{.ServiceModuleNamePlural}}Logic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.ServiceModuleNamePlural}}Logic) {{.ServiceModuleNamePlural}}(in *db.XRequestParams) (*db.Response, error) {
	params, err := conv.New(l.svcCtx.Config.Mode).ToOrmParams(in)
    if err != nil {
        return nil, response.NewMakeRpcRetErr(err, 2)
    }

	// 获取总数
	count, queryErr := l.svcCtx.{{.ServiceModuleNamePlural}}Model.Count(params)
	if queryErr != nil {
		return nil, response.NewMakeRpcRetErr(queryErr, 2)
	}

	if count <= 0 {
		return response.NewRpcResp[*db.Response]().Make(response.NewListResp[[]*{{.ModelName}}.Item]().Empty(), 3, func(data []byte) *db.Response {
			return &db.Response{Data: data}
		})
	}

	// 获取列表
	list, queryErr := l.svcCtx.{{.ServiceModuleNamePlural}}Model.List(params)
	if queryErr != nil {
		return nil, response.NewMakeRpcRetErr(queryErr, 2)
	}

	var records []*{{.ModelName}}.Item
	for _, item := range list {
		v, err := item.ConvToItem()
		if err != nil {
			return nil, response.NewMakeRpcRetErr(err, 2)
		}

		records = append(records, v)
	}

	return response.NewRpcResp[*db.Response]().Make(&response.ListResp[[]*{{.ModelName}}.Item]{
		List:  records,
		Count: count,
	}, 3, func(data []byte) *db.Response {
		return &db.Response{Data: data}
	})
}
