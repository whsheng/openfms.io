package model

import (
	"time"
)

// Vehicle 车辆信息
type Vehicle struct {
	ID              int       `json:"id" gorm:"primaryKey"`
	PlateNumber     string    `json:"plate_number" gorm:"column:plate_number;type:varchar(20);not null;uniqueIndex"`
	PlateColor      string    `json:"plate_color" gorm:"column:plate_color;type:varchar(10);default:'蓝色'"`
	VehicleType     string    `json:"vehicle_type,omitempty" gorm:"column:vehicle_type;type:varchar(50)"`
	Brand           string    `json:"brand,omitempty" gorm:"type:varchar(50)"`
	Model           string    `json:"model,omitempty" gorm:"type:varchar(50)"`
	Color           string    `json:"color,omitempty" gorm:"type:varchar(20)"`
	VIN             string    `json:"vin,omitempty" gorm:"type:varchar(17)"`
	EngineNo        string    `json:"engine_no,omitempty" gorm:"column:engine_no;type:varchar(30)"`
	OwnerName       string    `json:"owner_name,omitempty" gorm:"column:owner_name;type:varchar(100)"`
	OwnerPhone      string    `json:"owner_phone,omitempty" gorm:"column:owner_phone;type:varchar(20)"`
	OwnerIDCard     string    `json:"owner_idcard,omitempty" gorm:"column:owner_idcard;type:varchar(18)"`
	RegistrationNo  string    `json:"registration_no,omitempty" gorm:"column:registration_no;type:varchar(30)"`
	TransportNo     string    `json:"transport_no,omitempty" gorm:"column:transport_no;type:varchar(30)"`
	InsuranceNo     string    `json:"insurance_no,omitempty" gorm:"column:insurance_no;type:varchar(30)"`
	InsuranceExpire *time.Time `json:"insurance_expire,omitempty" gorm:"column:insurance_expire"`
	DeviceID        *string   `json:"device_id,omitempty" gorm:"column:device_id;type:varchar(20)"`
	Status          string    `json:"status" gorm:"type:varchar(20);default:'active'"`
	Remark          string    `json:"remark,omitempty" gorm:"type:text"`
	CreatedAt       time.Time `json:"created_at" gorm:"not null;default:now()"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"not null;default:now()"`
	
	// 关联
	Device *Device `json:"device,omitempty" gorm:"foreignKey:DeviceID;references:SimNo"`
}

func (Vehicle) TableName() string {
	return "vehicles"
}

// VehicleGroup 车辆分组
type VehicleGroup struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"type:varchar(50);not null"`
	Color       string    `json:"color" gorm:"type:varchar(20);default:'#1890ff'"`
	Description string    `json:"description,omitempty" gorm:"type:text"`
	CreatedBy   *int      `json:"created_by,omitempty" gorm:"column:created_by"`
	CreatedAt   time.Time `json:"created_at" gorm:"not null;default:now()"`
	
	// 关联
	VehicleCount int64 `json:"vehicle_count,omitempty" gorm:"-"`
}

func (VehicleGroup) TableName() string {
	return "vehicle_groups"
}

// VehicleGroupMember 车辆分组关联
type VehicleGroupMember struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	VehicleID int       `json:"vehicle_id" gorm:"column:vehicle_id;not null"`
	GroupID   int       `json:"group_id" gorm:"column:group_id;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"not null;default:now()"`
}

func (VehicleGroupMember) TableName() string {
	return "vehicle_group_members"
}

// DeviceVehicleHistory 设备车辆绑定历史
type DeviceVehicleHistory struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	DeviceID   string    `json:"device_id" gorm:"column:device_id;type:varchar(20);not null"`
	VehicleID  int       `json:"vehicle_id" gorm:"column:vehicle_id;not null"`
	Action     string    `json:"action" gorm:"type:varchar(20);not null"` // bind, unbind
	OperatedBy *int      `json:"operated_by,omitempty" gorm:"column:operated_by"`
	OperatedAt time.Time `json:"operated_at" gorm:"column:operated_at;not null;default:now()"`
	Remark     string    `json:"remark,omitempty" gorm:"type:text"`
}

func (DeviceVehicleHistory) TableName() string {
	return "device_vehicle_history"
}

// VehicleWithDevice 带设备信息的车辆
type VehicleWithDevice struct {
	Vehicle
	DeviceSimNo   *string `json:"device_sim_no,omitempty" gorm:"column:device_sim_no"`
	DeviceStatus  *string `json:"device_status,omitempty" gorm:"column:device_status"`
	DeviceOnline  *bool   `json:"device_online,omitempty" gorm:"column:device_online"`
}

// CreateVehicleRequest 创建车辆请求
type CreateVehicleRequest struct {
	PlateNumber     string     `json:"plate_number" binding:"required"`
	PlateColor      string     `json:"plate_color"`
	VehicleType     string     `json:"vehicle_type"`
	Brand           string     `json:"brand"`
	Model           string     `json:"model"`
	Color           string     `json:"color"`
	VIN             string     `json:"vin"`
	EngineNo        string     `json:"engine_no"`
	OwnerName       string     `json:"owner_name"`
	OwnerPhone      string     `json:"owner_phone"`
	OwnerIDCard     string     `json:"owner_idcard"`
	RegistrationNo  string     `json:"registration_no"`
	TransportNo     string     `json:"transport_no"`
	InsuranceNo     string     `json:"insurance_no"`
	InsuranceExpire *time.Time `json:"insurance_expire"`
	DeviceID        *string    `json:"device_id"`
	Remark          string     `json:"remark"`
}

// UpdateVehicleRequest 更新车辆请求
type UpdateVehicleRequest struct {
	PlateNumber     string     `json:"plate_number"`
	PlateColor      string     `json:"plate_color"`
	VehicleType     string     `json:"vehicle_type"`
	Brand           string     `json:"brand"`
	Model           string     `json:"model"`
	Color           string     `json:"color"`
	VIN             string     `json:"vin"`
	EngineNo        string     `json:"engine_no"`
	OwnerName       string     `json:"owner_name"`
	OwnerPhone      string     `json:"owner_phone"`
	OwnerIDCard     string     `json:"owner_idcard"`
	RegistrationNo  string     `json:"registration_no"`
	TransportNo     string     `json:"transport_no"`
	InsuranceNo     string     `json:"insurance_no"`
	InsuranceExpire *time.Time `json:"insurance_expire"`
	Status          string     `json:"status"`
	Remark          string     `json:"remark"`
}

// BindDeviceRequest 绑定设备请求
type BindDeviceRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Remark   string `json:"remark"`
}

// VehicleListQuery 车辆列表查询
type VehicleListQuery struct {
	PlateNumber string `form:"plate_number"`
	VehicleType string `form:"vehicle_type"`
	Status      string `form:"status"`
	DeviceID    string `form:"device_id"`
	GroupID     int    `form:"group_id"`
	Page        int    `form:"page,default=1"`
	PageSize    int    `form:"page_size,default=20"`
}
