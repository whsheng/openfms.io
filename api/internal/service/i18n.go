// 国际化 (i18n) 服务

package service

import (
	"context"
	"encoding/json"
	"fmt"
)

// I18nService 国际化服务
type I18nService struct {
	translations map[string]map[string]string // lang -> key -> value
	defaultLang  string
}

// NewI18nService 创建国际化服务
func NewI18nService() *I18nService {
	s := &I18nService{
		translations: make(map[string]map[string]string),
		defaultLang:  "zh-CN",
	}
	s.loadTranslations()
	return s
}

// loadTranslations 加载翻译
func (s *I18nService) loadTranslations() {
	// 中文翻译
	s.translations["zh-CN"] = map[string]string{
		"common.save":           "保存",
		"common.cancel":         "取消",
		"common.delete":         "删除",
		"common.edit":           "编辑",
		"common.create":         "创建",
		"common.search":         "搜索",
		"common.reset":          "重置",
		"common.submit":         "提交",
		"common.confirm":        "确认",
		"common.success":        "成功",
		"common.error":          "错误",
		"common.warning":        "警告",
		"common.info":           "信息",
		"common.loading":        "加载中...",
		"common.no_data":        "暂无数据",
		"common.load_more":      "加载更多",
		"common.refresh":        "刷新",
		
		"nav.dashboard":         "仪表盘",
		"nav.monitor":           "实时监控",
		"nav.devices":           "设备管理",
		"nav.vehicles":          "车辆管理",
		"nav.geofences":         "电子围栏",
		"nav.alarms":            "报警中心",
		"nav.reports":           "报表统计",
		"nav.video":             "视频监控",
		"nav.users":             "用户管理",
		"nav.settings":          "系统设置",
		
		"device.name":           "设备名称",
		"device.id":             "设备号",
		"device.status":         "设备状态",
		"device.online":         "在线",
		"device.offline":        "离线",
		"device.last_report":    "最后上报",
		
		"alarm.type":            "报警类型",
		"alarm.level":           "报警级别",
		"alarm.time":            "报警时间",
		"alarm.status":          "处理状态",
		"alarm.unread":          "未读",
		"alarm.read":            "已读",
		"alarm.resolved":        "已处理",
		
		"error.invalid_params":  "参数错误",
		"error.unauthorized":    "未授权",
		"error.forbidden":       "禁止访问",
		"error.not_found":       "资源不存在",
		"error.server_error":    "服务器错误",
	}
	
	// 英文翻译
	s.translations["en-US"] = map[string]string{
		"common.save":           "Save",
		"common.cancel":         "Cancel",
		"common.delete":         "Delete",
		"common.edit":           "Edit",
		"common.create":         "Create",
		"common.search":         "Search",
		"common.reset":          "Reset",
		"common.submit":         "Submit",
		"common.confirm":        "Confirm",
		"common.success":        "Success",
		"common.error":          "Error",
		"common.warning":        "Warning",
		"common.info":           "Info",
		"common.loading":        "Loading...",
		"common.no_data":        "No Data",
		"common.load_more":      "Load More",
		"common.refresh":        "Refresh",
		
		"nav.dashboard":         "Dashboard",
		"nav.monitor":           "Monitor",
		"nav.devices":           "Devices",
		"nav.vehicles":          "Vehicles",
		"nav.geofences":         "Geofences",
		"nav.alarms":            "Alarms",
		"nav.reports":           "Reports",
		"nav.video":             "Video",
		"nav.users":             "Users",
		"nav.settings":          "Settings",
		
		"device.name":           "Device Name",
		"device.id":             "Device ID",
		"device.status":         "Status",
		"device.online":         "Online",
		"device.offline":        "Offline",
		"device.last_report":    "Last Report",
		
		"alarm.type":            "Alarm Type",
		"alarm.level":           "Level",
		"alarm.time":            "Time",
		"alarm.status":          "Status",
		"alarm.unread":          "Unread",
		"alarm.read":            "Read",
		"alarm.resolved":        "Resolved",
		
		"error.invalid_params":  "Invalid Parameters",
		"error.unauthorized":    "Unauthorized",
		"error.forbidden":       "Forbidden",
		"error.not_found":       "Not Found",
		"error.server_error":    "Server Error",
	}
}

// T 翻译
func (s *I18nService) T(lang, key string) string {
	if lang == "" {
		lang = s.defaultLang
	}
	
	if trans, ok := s.translations[lang]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	
	// 回退到默认语言
	if trans, ok := s.translations[s.defaultLang]; ok {
		if val, ok := trans[key]; ok {
			return val
		}
	}
	
	return key
}

// GetSupportedLangs 获取支持的语言列表
func (s *I18nService) GetSupportedLangs() []map[string]string {
	return []map[string]string{
		{"code": "zh-CN", "name": "简体中文"},
		{"code": "en-US", "name": "English"},
	}
}

// GetTranslations 获取指定语言的所有翻译
func (s *I18nService) GetTranslations(lang string) map[string]string {
	if lang == "" {
		lang = s.defaultLang
	}
	
	if trans, ok := s.translations[lang]; ok {
		return trans
	}
	
	return s.translations[s.defaultLang]
}
