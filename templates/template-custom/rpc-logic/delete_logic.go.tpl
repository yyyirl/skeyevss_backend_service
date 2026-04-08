package {{.ServiceName}}logic

import (
	"context"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"

	"skeyevss/core/app/sev/db/db"
	"skeyevss/core/app/sev/db/internal/svc"
	"skeyevss/core/app/sev/db/pkg/conv"
	"skeyevss/core/pkg/response"
)

type {{.ServiceModuleNameSingular}}DeleteLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func New{{.ServiceModuleNameSingular}}DeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *{{.ServiceModuleNameSingular}}DeleteLogic {
	return &{{.ServiceModuleNameSingular}}DeleteLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *{{.ServiceModuleNameSingular}}DeleteLogic) {{.ServiceModuleNameSingular}}Delete(in *db.XRequestParams) (*db.Response, error) {
	params, err := conv.New(l.svcCtx.Config.Mode).ToOrmParams(in)
    if err != nil {
        return nil, response.NewMakeRpcRetErr(err, 2)
    }

	if err := response.NewMakeRpcRetErr(l.svcCtx.{{.ServiceModuleNamePlural}}Model.DeleteBy(params), 2); err != nil {
		return nil, response.NewMakeRpcRetErr(err, 2)
	}

	return &db.Response{Data: []byte(strconv.FormatBool(true))}, nil
}
