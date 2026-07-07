package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// Vendor 用于存储供应商信息，供模型引用
// Name 唯一，用于在模型中关联
// Icon 采用 @lobehub/icons 的图标名，前端可直接渲染
// Status 预留字段，1 表示启用
// 本表同样遵循 3NF 设计范式

type Vendor struct {
	Id          int            `json:"id"`
	Name        string         `json:"name" gorm:"size:128;not null;uniqueIndex:uk_vendor_name_delete_at,priority:1"`
	DisplayName string         `json:"display_name,omitempty" gorm:"type:varchar(256)"`
	Description string         `json:"description,omitempty" gorm:"type:text"`
	Icon        string         `json:"icon,omitempty" gorm:"type:varchar(128)"`
	Status      int            `json:"status" gorm:"default:1"`
	CreatedTime int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:uk_vendor_name_delete_at,priority:2"`
}

// Insert 创建新的供应商记录
func (v *Vendor) Insert() error {
	now := common.GetTimestamp()
	v.CreatedTime = now
	v.UpdatedTime = now
	return DB.Create(v).Error
}

// IsVendorNameDuplicated 检查供应商名称是否重复（排除自身 ID）
func IsVendorNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&Vendor{}).Where("name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

// Update 更新供应商记录
func (v *Vendor) Update() error {
	v.UpdatedTime = common.GetTimestamp()
	return DB.Save(v).Error
}

// Delete 软删除供应商
func (v *Vendor) Delete() error {
	return DB.Delete(v).Error
}

// GetVendorByName 根据 Name 查找供应商（用于幂等 upsert）
func GetVendorByName(name string) (*Vendor, error) {
	if name == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var v Vendor
	err := DB.Where("name = ?", name).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// UpsertVendorByName 幂等地按 Name 创建供应商。
// 已存在则不覆盖 DisplayName/Icon（保留管理员的手工修改）。
// 返回 (existed, error)：existed=true 表示记录已存在跳过写库。
func UpsertVendorByName(name, displayName, icon string) (bool, error) {
	if name == "" {
		return false, nil
	}
	existing, err := GetVendorByName(name)
	if err == nil && existing != nil {
		return true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	v := &Vendor{
		Name:        name,
		DisplayName: displayName,
		Icon:        icon,
		Status:      1,
	}
	if err := v.Insert(); err != nil {
		// 并发下如果被另一个请求先插了，也当作成功
		if existing2, e2 := GetVendorByName(name); e2 == nil && existing2 != nil {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

// GetAllVendorsForSync 拉取全部供应商（不分页，供 sync 使用）
func GetAllVendorsForSync() ([]*Vendor, error) {
	var vendors []*Vendor
	err := DB.Find(&vendors).Error
	return vendors, err
}

// GetVendorByID 根据 ID 获取供应商
func GetVendorByID(id int) (*Vendor, error) {
	var v Vendor
	err := DB.First(&v, id).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetAllVendors 获取全部供应商（分页）
func GetAllVendors(offset int, limit int) ([]*Vendor, error) {
	var vendors []*Vendor
	err := DB.Offset(offset).Limit(limit).Find(&vendors).Error
	return vendors, err
}

// SearchVendors 按关键字搜索供应商
func SearchVendors(keyword string, offset int, limit int) ([]*Vendor, int64, error) {
	db := DB.Model(&Vendor{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", like, like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var vendors []*Vendor
	if err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&vendors).Error; err != nil {
		return nil, 0, err
	}
	return vendors, total, nil
}
