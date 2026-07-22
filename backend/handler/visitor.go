package handler

import (
	"github.com/kingwrcy/moments/db"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
)

const (
	defaultVisitorPageSize = 30
	maxVisitorPageSize     = 100
)

type VisitorHandler struct {
	base BaseHandler
}

type listVisitorsReq struct {
	Page int `json:"page,omitempty"`
	Size int `json:"size,omitempty"`
}

type visitorListResp struct {
	List    []db.Visitor `json:"list"`
	Total   int64        `json:"total"`
	HasNext bool         `json:"hasNext"`
}

func NewVisitorHandler(injector do.Injector) *VisitorHandler {
	return &VisitorHandler{do.MustInvoke[BaseHandler](injector)}
}

// List returns observed visitor metadata to the site administrator only.
// VisitorID remains excluded from JSON by the database model; it is an
// internal HttpOnly-cookie identifier and is never exposed through this API.
func (v VisitorHandler) List(c echo.Context) error {
	context := c.(CustomContext)
	currentUser := context.CurrentUser()
	if currentUser == nil || currentUser.Id != 1 {
		return FailRespWithMsg(c, Fail, "仅管理员可查看访客记录")
	}

	var req listVisitorsReq
	if err := c.Bind(&req); err != nil {
		return FailResp(c, ParamError)
	}
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Size < 1 {
		req.Size = defaultVisitorPageSize
	}
	if req.Size > maxVisitorPageSize {
		req.Size = maxVisitorPageSize
	}

	var (
		list  []db.Visitor
		total int64
	)
	if err := v.base.db.Model(&db.Visitor{}).Count(&total).Error; err != nil {
		v.base.log.Error().Err(err).Msg("查询访客记录失败")
		return FailRespWithMsg(c, Fail, "查询访客记录失败")
	}
	if err := v.base.db.Order("lastSeenAt DESC, id DESC").Limit(req.Size).Offset((req.Page - 1) * req.Size).Find(&list).Error; err != nil {
		v.base.log.Error().Err(err).Msg("读取访客记录失败")
		return FailRespWithMsg(c, Fail, "读取访客记录失败")
	}

	return SuccessResp(c, visitorListResp{
		List:    list,
		Total:   total,
		HasNext: int64(req.Page*req.Size) < total,
	})
}
