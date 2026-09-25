package model

import "time"

// SysUser is a local operation-admin account (independent from new-api users).
type SysUser struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:64;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	RealName     string    `json:"real_name" gorm:"size:64"`
	Phone        string    `json:"phone" gorm:"size:32"`
	Email        string    `json:"email" gorm:"size:128"`
	Status       string    `json:"status" gorm:"size:16;index"` // active / inactive
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Roles        []SysRole `json:"roles,omitempty" gorm:"many2many:sys_user_roles;"`
}

func (SysUser) TableName() string { return "sys_users" }

type SysRole struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Code      string    `json:"code" gorm:"uniqueIndex;size:64;not null"`
	Name      string    `json:"name" gorm:"size:64;not null"`
	Status    string    `json:"status" gorm:"size:16"`
	Remark    string    `json:"remark" gorm:"size:255"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Menus     []SysMenu `json:"menus,omitempty" gorm:"many2many:sys_role_menus;"`
}

func (SysRole) TableName() string { return "sys_roles" }

// SysMenu: menuType M=目录 C=菜单 F=按钮
type SysMenu struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	ParentID   *uint     `json:"parent_id" gorm:"index"`
	Name       string    `json:"name" gorm:"size:64;not null"`
	Path       string    `json:"path" gorm:"size:128"`
	Permission string    `json:"permission" gorm:"size:128;index"`
	Icon       string    `json:"icon" gorm:"size:64"`
	ParentIcon string    `json:"parent_icon" gorm:"size:64"`
	Subsystem  string    `json:"subsystem" gorm:"size:32;index"` // finance / ops / system / common
	MenuType   string    `json:"menu_type" gorm:"size:8"`        // M / C / F
	MenuGroup  string    `json:"menu_group" gorm:"size:64"`
	Sort       int       `json:"sort"`
	Status     string    `json:"status" gorm:"size:16"`
	Hidden     bool      `json:"hidden"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Children   []SysMenu `json:"children,omitempty" gorm:"-"`
}

func (SysMenu) TableName() string { return "sys_menus" }
