package {{.ServiceName}}logic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/db/db"
	"skeyevss/core/app/sev/db/internal/svc"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/pkg/response"
	"skeyevss/core/repositories/models/{{.ModelName}}"
)

type {{.ServiceModuleNameSingular}}UpdateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.ServiceModuleNameSingular}}UpdateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.ServiceModuleNameSingular}}UpdateLogic {
	return &{{.ServiceModuleNameSingular}}UpdateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.ServiceModuleNameSingular}}UpdateLogic) {{.ServiceModuleNameSingular}}Update(in *db.XRequestParams) (*db.Response, error) {
	params, err := conv.New(l.svcCtx.Config.Mode).ToOrmParams(in)
    if err != nil {
        return nil, response.NewMakeRpcRetErr(err, 2)
    }

	record, err := {{.ModelName}}.NewItem().CheckMap(params.DataRecord)
	if err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	if err := response.NewMakeRpcRetErr(l.svcCtx.{{.ServiceModuleNamePlural}}Model.UpdateWithParams(record, params), 2); err != nil {
        return nil, response.NewMakeRpcRetErr(err, 2)
    }

    return &db.Response{Data: []byte(strconv.FormatBool(true))}, nil
}
