package handler

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/kingwrcy/moments/db"
	"github.com/kingwrcy/moments/vo"
	"github.com/labstack/echo/v4"
	"github.com/samber/do/v2"
	"gorm.io/gorm"
)

type SysConfigHandler struct {
	base BaseHandler
}

func hasConfigField(content, field string) bool {
	return strings.Contains(content, `"`+field+`"`)
}

// interactionDefaults keeps existing installations on their previous behaviour
// until the new, separate interaction switches are explicitly saved.
func interactionDefaults(content string, legacyVisible, enableLike, showLikeCount, enableComment, showComments bool) (bool, bool, bool, bool) {
	if !hasConfigField(content, "showVisitorInteractions") {
		legacyVisible = true
	}
	if !hasConfigField(content, "enableLike") {
		enableLike = legacyVisible
	}
	if !hasConfigField(content, "showVisitorLikeCount") {
		showLikeCount = enableLike && legacyVisible
	}
	if !hasConfigField(content, "showVisitorComments") {
		showComments = enableComment && legacyVisible
	}
	if !enableLike {
		showLikeCount = false
	}
	if !enableComment {
		showComments = false
	}
	return enableLike, showLikeCount, enableComment, showComments
}

func applySysConfigDefaults(content string, result *vo.SysConfigVO) {
	result.EnableLike, result.ShowVisitorLikeCount, result.EnableComment, result.ShowVisitorComments = interactionDefaults(
		content,
		result.ShowVisitorInteractions,
		result.EnableLike,
		result.ShowVisitorLikeCount,
		result.EnableComment,
		result.ShowVisitorComments,
	)
	if !hasConfigField(content, "enableExternalAccess") {
		result.EnableExternalAccess = true
	}
}

func applyFullSysConfigDefaults(content string, result *vo.FullSysConfigVO) {
	result.EnableLike, result.ShowVisitorLikeCount, result.EnableComment, result.ShowVisitorComments = interactionDefaults(
		content,
		result.ShowVisitorInteractions,
		result.EnableLike,
		result.ShowVisitorLikeCount,
		result.EnableComment,
		result.ShowVisitorComments,
	)
	if !hasConfigField(content, "enableExternalAccess") {
		result.EnableExternalAccess = true
	}
}

func NewSysConfigHandler(injector do.Injector) *SysConfigHandler {
	return &SysConfigHandler{do.MustInvoke[BaseHandler](injector)}
}

// GetConfig godoc
//
//	@Tags			SysConfig
//	@Summary		获取系统设置(部分不敏感的)
//	@Description	敏感信息不返回,包括各种key密钥
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	vo.SysConfigVO
//	@Router			/sysConfig/get [post]
func (s SysConfigHandler) GetConfig(c echo.Context) error {
	var (
		config db.SysConfig
		result vo.SysConfigVO
	)

	if err := s.base.db.First(&config).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return SuccessResp(c, h{})
	}
	err := json.Unmarshal([]byte(config.Content), &result)
	if err != nil {
		return FailRespWithMsg(c, Fail, "读取系统配置异常")
	}
	applySysConfigDefaults(config.Content, &result)
	result.Version = s.base.cfg.Version
	result.CommitId = s.base.cfg.CommitId

	suffix := result.S3.ThumbnailSuffix
	result.S3 = vo.S3VO{
		ThumbnailSuffix: suffix,
	}
	return SuccessResp(c, result)
}

// GetFullConfig godoc
//
//	@Tags		SysConfig
//	@Summary	获取系统设置(完整的)
//	@Accept		json
//	@Produce	json
//	@Param		x-api-token	header		string	true	"登录TOKEN"
//	@Success	200			{object}	vo.FullSysConfigVO
//	@Success	200
//	@Router		/api/sysConfig/get [post]
func (s SysConfigHandler) GetFullConfig(c echo.Context) error {
	var (
		config db.SysConfig
		result vo.FullSysConfigVO
	)

	context := c.(CustomContext)
	currentUser := context.CurrentUser()
	if currentUser == nil || currentUser.Id != 1 {
		return FailRespWithMsg(c, Fail, "需要先登录")
	}
	if err := s.base.db.First(&config).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return SuccessResp(c, h{})
	}
	err := json.Unmarshal([]byte(config.Content), &result)
	if err != nil {
		return FailRespWithMsg(c, Fail, "读取系统配置异常")
	}
	applyFullSysConfigDefaults(config.Content, &result)
	result.Version = s.base.cfg.Version
	result.CommitId = s.base.cfg.CommitId
	return SuccessResp(c, result)
}

// SaveConfig godoc
//
//	@Tags		SysConfig
//	@Summary	保存系统设置
//	@Accept		json
//	@Produce	json
//	@Param		object		body	vo.FullSysConfigVO	true	"保存系统设置"
//	@Param		x-api-token	header	string				true	"登录TOKEN"
//	@Success	200
//	@Router		/api/sysConfig/save [post]
func (s SysConfigHandler) SaveConfig(c echo.Context) error {
	var (
		config db.SysConfig
		result vo.FullSysConfigVO
	)
	context := c.(CustomContext)
	currentUser := context.CurrentUser()
	if currentUser == nil || currentUser.Id != 1 {
		return FailRespWithMsg(c, Fail, "需要先登录")
	}

	if err := c.Bind(&result); err != nil {
		s.base.log.Info().Msgf("保存配置错误,%s", err)
		return FailResp(c, ParamError)
	}
	// A count or comment list without its corresponding interaction does not
	// make sense for visitors. Enforce the dependency server-side as well as in
	// the settings UI so manually crafted requests cannot leave an invalid state.
	if !result.EnableLike {
		result.ShowVisitorLikeCount = false
	}
	if !result.EnableComment {
		result.ShowVisitorComments = false
	}

	data, err := json.Marshal(result)
	if err != nil {
		return FailRespWithMsg(c, Fail, "读取系统配置异常")
	}

	if err := s.base.db.First(&config).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		config.Content = string(data)
		if err = s.base.db.Save(&config).Error; err != nil {
			return FailRespWithMsg(c, Fail, "保存系统配置异常")
		}
	} else {
		config.Content = string(data)
		if err = s.base.db.Updates(&config).Error; err != nil {
			return FailRespWithMsg(c, Fail, "保存系统配置异常")
		}
	}
	s.base.db.Table("User").Where("id=?", 1).Update("username", result.AdminUserName)
	return SuccessResp(c, h{})
}
