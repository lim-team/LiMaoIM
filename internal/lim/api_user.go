package lim

import (
	"net/http"

	"github.com/lim-team/LiMaoIM/pkg/lmhttp"
	"github.com/lim-team/LiMaoIM/pkg/lmproto"
	"github.com/pkg/errors"
	"github.com/tangtaoit/limnet/pkg/limlog"
	"go.uber.org/zap"
)

// UserAPI 用户相关API
type UserAPI struct {
	limlog.Log
	l *LiMao
}

// NewUserAPI NewUserAPI
func NewUserAPI(l *LiMao) *UserAPI {
	return &UserAPI{
		Log: limlog.NewLIMLog("UserAPI"),
		l:   l,
	}
}

// Route 用户相关路由配置
func (u *UserAPI) Route(r *lmhttp.LMHttp) {
	// 更新用户token
	r.POST("/user/token", u.updateToken)
	// 获取用户在线状态
	r.POST("/user/onlinestatus", u.getOnlineStatus)

	r.POST("/systemuids_add", u.systemUIDsAdd)       // 添加系统uid
	r.POST("/systemuids_remove", u.systemUIDsRemove) // 移除系统uid
}

func (u *UserAPI) getOnlineStatus(c *lmhttp.Context) {
	var uids []string
	if err := c.BindJSON(&uids); err != nil {
		c.ResponseError(err)
		return
	}
	c.JSON(http.StatusOK, u.l.clientManager.GetOnlineUIDs(uids))
}

// 更新用户的token
func (u *UserAPI) updateToken(c *lmhttp.Context) {
	var req UpdateTokenReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(err)
		return
	}
	if err := req.Check(); err != nil {
		c.ResponseError(err)
		return
	}
	u.Debug("req", zap.Any("req", req))

	err := u.l.store.UpdateUserToken(req.UID, req.DeviceFlag, req.DeviceLevel, req.Token)
	if err != nil {
		u.Error("更新用户token失败！", zap.Error(err))
		c.ResponseError(errors.Wrap(err, "更新用户token失败！"))
		return
	}

	// 如果存在旧连接，则发起踢出请求
	oldClient := u.l.clientManager.GetClientWith(req.UID, req.DeviceFlag)
	if oldClient != nil {
		oldClient.WritePacket(&lmproto.DisconnectPacket{
			ReasonCode: 0,
			Reason:     "账号在其他设备上登录",
		})
	}
	// 创建或更新个人频道
	err = u.l.channelManager.CreateOrUpdatePersonChannel(req.UID)
	if err != nil {
		u.Error("创建个人频道失败！", zap.Error(err))
		c.ResponseError(errors.New("创建个人频道失败！"))
		return
	}
	c.ResponseOK()
}

// 添加系统uid
func (u *UserAPI) systemUIDsAdd(c *lmhttp.Context) {
	var req struct {
		UIDs []string `json:"uids"`
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(req.UIDs) > 0 {
		for _, uid := range req.UIDs {
			u.l.systemUIDManager.AddSystemUID(uid)
		}
	}
	c.ResponseOK()

}

// 移除系统uid
func (u *UserAPI) systemUIDsRemove(c *lmhttp.Context) {
	var req struct {
		UIDs []string `json:"uids"`
	}
	if err := c.BindJSON(&req); err != nil {
		u.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(req.UIDs) > 0 {
		for _, uid := range req.UIDs {
			u.l.systemUIDManager.RemoveSystemUID(uid)
		}
	}
	c.ResponseOK()
}

// UpdateTokenReq 更新token请求
type UpdateTokenReq struct {
	UID         string              `json:"uid"`
	Token       string              `json:"token"`
	DeviceFlag  lmproto.DeviceFlag  `json:"device_flag"`
	DeviceLevel lmproto.DeviceLevel `json:"device_level"` // 设备等级 0.为从设备 1.为主设备
}

// Check 检查输入
func (u UpdateTokenReq) Check() error {
	if u.UID == "" {
		return errors.New("uid不能为空！")
	}
	// if len(u.UID) > 32 {
	// 	return errors.New("uid不能大于32位")
	// }
	if u.Token == "" {
		return errors.New("token不能为空！")
	}
	// if len(u.PublicKey) <= 0 {
	// 	return errors.New("用户RSA公钥不能为空！")
	// }
	// if len(u.Token) > 32 {
	// 	return errors.New("token不能大于32位")
	// }
	return nil
}
