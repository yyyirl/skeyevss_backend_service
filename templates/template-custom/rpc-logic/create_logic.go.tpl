package {{.ServiceName}}logic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/db/db"
	"skeyevss/core/app/sev/db/internal/svc"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/{{.ModelName}}"
)

type {{.ServiceModuleNameSingular}}CreateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.ServiceModuleNameSingular}}CreateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.ServiceModuleNameSingular}}CreateLogic {
	return &{{.ServiceModuleNameSingular}}CreateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.ServiceModuleNameSingular}}CreateLogic) {{.ServiceModuleNameSingular}}Create(in *db.MapReq) (*db.Response, error) {
	record, err := {{.ModelName}}.NewItem().MapToModel(in.Data.AsMap())
	if err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	data, err := record.ConvToModel(func(item *{{.ModelName}}.Item) *{{.ModelName}}.Item {
		return item
	})
	if err != nil || data == nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	res, err := l.svcCtx.{{.ServiceModuleNamePlural}}Model.Add(*data)
	if err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	return &db.Response{Data: []byte(strconv.Itoa(int(res.ID)))}, nil
}
