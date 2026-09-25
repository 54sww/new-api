package rbac

import (
	"log"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"gorm.io/gorm"
)

type seedMenu struct {
	Key        string
	ParentKey  string
	Name       string
	Path       string
	Permission string
	Icon       string
	ParentIcon string
	Subsystem  string
	MenuType   string
	MenuGroup  string
	Sort       int
}

var seedMenus = []seedMenu{
	{Key: "home", Name: "首页", Path: "/dashboard", Permission: "common:home", Icon: "HomeFilled", Subsystem: "common", MenuType: "C", Sort: 1},

	{Key: "finance_group", Name: "财务对账", Icon: "Wallet", ParentIcon: "Wallet", Subsystem: "finance", MenuType: "M", MenuGroup: "财务对账", Sort: 10},
	{Key: "finance_reconcile", ParentKey: "finance_group", Name: "实时对账", Path: "/finance/reconcile", Permission: "finance:reconcile:view", Icon: "DataAnalysis", ParentIcon: "Wallet", Subsystem: "finance", MenuType: "C", MenuGroup: "财务对账", Sort: 11},
	{Key: "finance_daily", ParentKey: "finance_group", Name: "消耗日报", Path: "/finance/daily", Permission: "finance:daily:view", Icon: "Calendar", ParentIcon: "Wallet", Subsystem: "finance", MenuType: "C", MenuGroup: "财务对账", Sort: 12},
	{Key: "finance_sync_settings", ParentKey: "finance_group", Name: "自动同步", Path: "/finance/sync-settings", Permission: "finance:sync-settings:view", Icon: "Timer", ParentIcon: "Wallet", Subsystem: "finance", MenuType: "C", MenuGroup: "财务对账", Sort: 13},
	{Key: "finance_sync_settings_edit", ParentKey: "finance_sync_settings", Name: "编辑自动同步", Permission: "finance:sync-settings:edit", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 14},
	{Key: "finance_sync", ParentKey: "finance_group", Name: "立即同步", Permission: "finance:reconcile:sync", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 15},
	{Key: "finance_balance_cfg", ParentKey: "finance_group", Name: "渠道配置", Permission: "finance:balance:config", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 16},
	{Key: "finance_balance_refresh", ParentKey: "finance_group", Name: "刷新余额", Permission: "finance:balance:refresh", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 17},
	{Key: "finance_opening", ParentKey: "finance_group", Name: "设期初", Permission: "finance:opening:set", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 18},
	{Key: "finance_recharge_create", ParentKey: "finance_group", Name: "登记充值", Permission: "finance:recharge:create", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 19},
	{Key: "finance_recharge_delete", ParentKey: "finance_group", Name: "删除充值", Permission: "finance:recharge:delete", Subsystem: "finance", MenuType: "F", MenuGroup: "财务对账", Sort: 20},

	{Key: "ops_group", Name: "运维监控", Icon: "SetUp", ParentIcon: "SetUp", Subsystem: "ops", MenuType: "M", MenuGroup: "运维监控", Sort: 20},
	{Key: "ops_overview", ParentKey: "ops_group", Name: "运维总览", Path: "/ops/overview", Permission: "ops:overview:view", Icon: "Monitor", ParentIcon: "SetUp", Subsystem: "ops", MenuType: "C", MenuGroup: "运维监控", Sort: 21},

	{Key: "sys_group", Name: "权限管理", Icon: "Lock", ParentIcon: "Lock", Subsystem: "system", MenuType: "M", MenuGroup: "权限管理", Sort: 30},
	{Key: "sys_user", ParentKey: "sys_group", Name: "用户管理", Path: "/system/users", Permission: "system:user:list", Icon: "User", ParentIcon: "Lock", Subsystem: "system", MenuType: "C", MenuGroup: "权限管理", Sort: 31},
	{Key: "sys_user_create", ParentKey: "sys_user", Name: "新增用户", Permission: "system:user:create", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 32},
	{Key: "sys_user_edit", ParentKey: "sys_user", Name: "编辑用户", Permission: "system:user:edit", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 33},
	{Key: "sys_user_delete", ParentKey: "sys_user", Name: "删除用户", Permission: "system:user:delete", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 34},
	{Key: "sys_role", ParentKey: "sys_group", Name: "角色管理", Path: "/system/roles", Permission: "system:role:list", Icon: "UserFilled", ParentIcon: "Lock", Subsystem: "system", MenuType: "C", MenuGroup: "权限管理", Sort: 35},
	{Key: "sys_role_create", ParentKey: "sys_role", Name: "新增角色", Permission: "system:role:create", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 36},
	{Key: "sys_role_edit", ParentKey: "sys_role", Name: "编辑角色", Permission: "system:role:edit", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 37},
	{Key: "sys_role_delete", ParentKey: "sys_role", Name: "删除角色", Permission: "system:role:delete", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 38},
	{Key: "sys_role_assign", ParentKey: "sys_role", Name: "分配菜单", Permission: "system:role:assign", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 39},
	{Key: "sys_menu", ParentKey: "sys_group", Name: "菜单管理", Path: "/system/menus", Permission: "system:menu:list", Icon: "Menu", ParentIcon: "Lock", Subsystem: "system", MenuType: "C", MenuGroup: "权限管理", Sort: 40},
	{Key: "sys_menu_create", ParentKey: "sys_menu", Name: "新增菜单", Permission: "system:menu:create", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 41},
	{Key: "sys_menu_edit", ParentKey: "sys_menu", Name: "编辑菜单", Permission: "system:menu:edit", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 42},
	{Key: "sys_menu_delete", ParentKey: "sys_menu", Name: "删除菜单", Permission: "system:menu:delete", Subsystem: "system", MenuType: "F", MenuGroup: "权限管理", Sort: 43},

	{Key: "notify_group", Name: "通知管理", Icon: "Bell", ParentIcon: "Bell", Subsystem: "system", MenuType: "M", MenuGroup: "通知管理", Sort: 50},
	{Key: "notify_channel", ParentKey: "notify_group", Name: "通知渠道", Path: "/system/notify-channels", Permission: "system:notify-channel:list", Icon: "Message", ParentIcon: "Bell", Subsystem: "system", MenuType: "C", MenuGroup: "通知管理", Sort: 51},
	{Key: "notify_channel_create", ParentKey: "notify_channel", Name: "新增通知渠道", Permission: "system:notify-channel:create", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 52},
	{Key: "notify_channel_edit", ParentKey: "notify_channel", Name: "编辑通知渠道", Permission: "system:notify-channel:edit", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 53},
	{Key: "notify_channel_delete", ParentKey: "notify_channel", Name: "删除通知渠道", Permission: "system:notify-channel:delete", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 54},
	{Key: "notify_group_page", ParentKey: "notify_group", Name: "通知组", Path: "/system/notify-groups", Permission: "system:notify-group:list", Icon: "Collection", ParentIcon: "Bell", Subsystem: "system", MenuType: "C", MenuGroup: "通知管理", Sort: 55},
	{Key: "notify_group_create", ParentKey: "notify_group_page", Name: "新增通知组", Permission: "system:notify-group:create", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 56},
	{Key: "notify_group_edit", ParentKey: "notify_group_page", Name: "编辑通知组", Permission: "system:notify-group:edit", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 57},
	{Key: "notify_group_delete", ParentKey: "notify_group_page", Name: "删除通知组", Permission: "system:notify-group:delete", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 58},
	{Key: "notify_rule", ParentKey: "notify_group", Name: "通知规则", Path: "/system/notify-rules", Permission: "system:notify-rule:list", Icon: "AlarmClock", ParentIcon: "Bell", Subsystem: "system", MenuType: "C", MenuGroup: "通知管理", Sort: 59},
	{Key: "notify_rule_create", ParentKey: "notify_rule", Name: "新增通知规则", Permission: "system:notify-rule:create", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 60},
	{Key: "notify_rule_edit", ParentKey: "notify_rule", Name: "编辑通知规则", Permission: "system:notify-rule:edit", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 61},
	{Key: "notify_rule_delete", ParentKey: "notify_rule", Name: "删除通知规则", Permission: "system:notify-rule:delete", Subsystem: "system", MenuType: "F", MenuGroup: "通知管理", Sort: 62},
}

// Seed idempotently creates default menus, admin role, and admin user.
func Seed(db *gorm.DB, adminPassword string) error {
	idByKey := map[string]uint{}

	for _, sm := range seedMenus {
		var existing model.SysMenu
		q := db.Where("name = ? AND subsystem = ? AND menu_type = ?", sm.Name, sm.Subsystem, sm.MenuType)
		if sm.Permission != "" {
			q = db.Where("permission = ?", sm.Permission)
		} else if sm.Path != "" {
			q = db.Where("path = ? AND menu_type = ?", sm.Path, sm.MenuType)
		}
		err := q.First(&existing).Error
		var parentID *uint
		if sm.ParentKey != "" {
			if pid, ok := idByKey[sm.ParentKey]; ok {
				parentID = &pid
			}
		}
		if err == gorm.ErrRecordNotFound {
			m := model.SysMenu{
				ParentID:   parentID,
				Name:       sm.Name,
				Path:       sm.Path,
				Permission: sm.Permission,
				Icon:       sm.Icon,
				ParentIcon: sm.ParentIcon,
				Subsystem:  sm.Subsystem,
				MenuType:   sm.MenuType,
				MenuGroup:  sm.MenuGroup,
				Sort:       sm.Sort,
				Status:     "active",
			}
			if err := db.Create(&m).Error; err != nil {
				return err
			}
			idByKey[sm.Key] = m.ID
			continue
		}
		if err != nil {
			return err
		}
		updates := map[string]interface{}{
			"parent_id":   parentID,
			"name":        sm.Name,
			"path":        sm.Path,
			"permission":  sm.Permission,
			"icon":        sm.Icon,
			"parent_icon": sm.ParentIcon,
			"subsystem":   sm.Subsystem,
			"menu_type":   sm.MenuType,
			"menu_group":  sm.MenuGroup,
			"sort":        sm.Sort,
			"status":      "active",
		}
		if err := db.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		idByKey[sm.Key] = existing.ID
	}

	var adminRole model.SysRole
	err := db.Where("code = ?", "admin").First(&adminRole).Error
	if err == gorm.ErrRecordNotFound {
		adminRole = model.SysRole{Code: "admin", Name: "超级管理员", Status: "active", Remark: "拥有全部权限"}
		if err := db.Create(&adminRole).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	var allMenus []model.SysMenu
	if err := db.Find(&allMenus).Error; err != nil {
		return err
	}
	if err := db.Model(&adminRole).Association("Menus").Replace(allMenus); err != nil {
		return err
	}

	var adminUser model.SysUser
	err = db.Where("username = ?", "admin").First(&adminUser).Error
	if err == gorm.ErrRecordNotFound {
		hash, err := HashPassword(adminPassword)
		if err != nil {
			return err
		}
		adminUser = model.SysUser{
			Username:     "admin",
			PasswordHash: hash,
			RealName:     "管理员",
			Status:       "active",
		}
		if err := db.Create(&adminUser).Error; err != nil {
			return err
		}
		log.Printf("seeded local admin user (username=admin)")
	} else if err != nil {
		return err
	}
	if err := db.Model(&adminUser).Association("Roles").Replace([]model.SysRole{adminRole}); err != nil {
		return err
	}

	// finance viewer role example (no balance config)
	var viewer model.SysRole
	err = db.Where("code = ?", "finance_viewer").First(&viewer).Error
	if err == gorm.ErrRecordNotFound {
		viewer = model.SysRole{Code: "finance_viewer", Name: "财务只读", Status: "active", Remark: "可看对账，不可改余额配置"}
		if err := db.Create(&viewer).Error; err != nil {
			return err
		}
		allow := []string{
			"common:home",
			"finance:reconcile:view",
			"finance:daily:view",
			"finance:sync-settings:view",
			"finance:balance:refresh",
			"ops:overview:view",
		}
		var menus []model.SysMenu
		db.Where("permission IN ?", allow).Find(&menus)
		// also attach parent directories
		var parents []model.SysMenu
		db.Where("menu_type = ? AND subsystem IN ?", "M", []string{"finance", "ops"}).Find(&parents)
		menus = append(menus, parents...)
		_ = db.Model(&viewer).Association("Menus").Replace(menus)
	}

	return nil
}
