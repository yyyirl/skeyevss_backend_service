package {{.PkgModuleName}}

import (
	"net/http"

	"skeyevss/core/app/sev/backend/internal/logic/{{.LogicDir}}"
	"skeyevss/core/app/sev/backend/internal/svc"
	"skeyevss/core/common/source/permissions"
	"skeyevss/core/localization"
	"skeyevss/core/pkg/common"
	"skeyevss/core/pkg/contextx"
	"skeyevss/core/pkg/orm"
	"skeyevss/core/pkg/response"
)

func UpdateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ctx = r.Context()
		if err := permissions.New(ctx).Authentication(contextx.GetSuperState(ctx), "permissions.TODO", contextx.GetPermissionIds(ctx)); err != nil {
			response.New().RequestError(ctx, w, response.MakeError(response.NewHttpRespMessage().Err(err), localization.M1006))
			return
		}

		var req orm.ReqParams
		if err := common.Parse(r, &req); err != nil {
			response.New().RequestError(ctx, w, response.MakeError(response.NewHttpRespMessage().Err(err), localization.M0001))
			return
		}

		if err := {{.ModelName}}.NewUpdateLogic(ctx, svcCtx).Update(&req); err != nil {
			response.New().RequestError(ctx, w, err)
			return
		}

		response.New().Success(ctx, w, nil)
	}
}
