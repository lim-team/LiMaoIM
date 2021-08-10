package lim

import (
	"fmt"
	"net/http"

	"github.com/lim-team/LiMaoIM/pkg/network"
	"github.com/lim-team/LiMaoIM/pkg/util"
)

// IDatasource 数据源第三方应用可以提供
type IDatasource interface {
	// 获取订阅者
	GetSubscribers(channelID string, channelType uint8) ([]string, error)
	// 获取黑名单
	GetBlacklist(channelID string, channelType uint8) ([]string, error)
	// 获取白名单
	GetWhitelist(channelID string, channelType uint8) ([]string, error)
	// 获取系统账号的uid集合 系统账号可以给任何人发消息
	GetSystemUIDs() ([]string, error)
}

// Datasource Datasource
type Datasource struct {
	l *LiMao
}

// NewDatasource 创建一个数据源
func NewDatasource(l *LiMao) IDatasource {

	return &Datasource{
		l: l,
	}
}

// GetSubscribers 获取频道的订阅者
func (d *Datasource) GetSubscribers(channelID string, channelType uint8) ([]string, error) {

	result, err := d.requestCMD("getSubscribers", map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
	})
	if err != nil {
		return nil, err
	}
	var subscribers []string
	err = util.ReadJSONByByte([]byte(result), &subscribers)
	if err != nil {
		return nil, err
	}
	return subscribers, nil
}

// GetBlacklist 获取频道的黑名单
func (d *Datasource) GetBlacklist(channelID string, channelType uint8) ([]string, error) {

	result, err := d.requestCMD("getBlacklist", map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
	})
	if err != nil {
		return nil, err
	}

	var blacklists []string
	err = util.ReadJSONByByte([]byte(result), &blacklists)
	if err != nil {
		return nil, err
	}
	return blacklists, nil
}

// GetWhitelist 获取频道的白明单
func (d *Datasource) GetWhitelist(channelID string, channelType uint8) ([]string, error) {

	result, err := d.requestCMD("getWhitelist", map[string]interface{}{
		"channel_id":   channelID,
		"channel_type": channelType,
	})
	if err != nil {
		return nil, err
	}
	var whitelists []string
	err = util.ReadJSONByByte([]byte(result), &whitelists)
	if err != nil {
		return nil, err
	}
	return whitelists, nil
}

// GetSystemUIDs 获取系统账号
func (d *Datasource) GetSystemUIDs() ([]string, error) {
	result, err := d.requestCMD("getSystemUIDs", map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	var uids []string
	err = util.ReadJSONByByte([]byte(result), &uids)
	if err != nil {
		return nil, err
	}
	return uids, nil
}

func (d *Datasource) requestCMD(cmd string, param map[string]interface{}) (string, error) {
	dataMap := map[string]interface{}{
		"cmd": cmd,
	}
	if param != nil {
		dataMap["data"] = param
	}
	resp, err := network.Post(d.l.opts.Datasource, []byte(util.ToJSON(dataMap)), nil)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http状态码错误！[%d]", resp.StatusCode)
	}

	return resp.Body, nil
}
