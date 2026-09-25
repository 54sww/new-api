package rbac

import (
	"fmt"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func (s *Service) IsSuperAdmin(userID uint, authSource string) bool {
	if authSource == "newapi" {
		return true
	}
	var n int64
	s.DB.Table("sys_user_roles").
		Joins("JOIN sys_roles ON sys_roles.id = sys_user_roles.sys_role_id").
		Where("sys_user_roles.sys_user_id = ? AND sys_roles.code = ? AND sys_roles.status = ?", userID, "admin", "active").
		Count(&n)
	return n > 0
}

func (s *Service) UserPermissions(userID uint, authSource string) ([]string, error) {
	if s.IsSuperAdmin(userID, authSource) {
		var all []string
		err := s.DB.Model(&model.SysMenu{}).
			Where("permission <> '' AND status = ?", "active").
			Distinct().Pluck("permission", &all).Error
		return all, err
	}
	var perms []string
	err := s.DB.Table("sys_menus").
		Select("DISTINCT sys_menus.permission").
		Joins("JOIN sys_role_menus ON sys_role_menus.sys_menu_id = sys_menus.id").
		Joins("JOIN sys_user_roles ON sys_user_roles.sys_role_id = sys_role_menus.sys_role_id").
		Joins("JOIN sys_roles ON sys_roles.id = sys_user_roles.sys_role_id").
		Where("sys_user_roles.sys_user_id = ? AND sys_roles.status = ? AND sys_menus.status = ? AND sys_menus.permission <> ''",
			userID, "active", "active").
		Pluck("sys_menus.permission", &perms).Error
	return perms, err
}

func (s *Service) UserMenuPaths(userID uint, authSource string) ([]string, error) {
	if s.IsSuperAdmin(userID, authSource) {
		var paths []string
		err := s.DB.Model(&model.SysMenu{}).
			Where("menu_type = ? AND path <> '' AND status = ? AND hidden = ?", "C", "active", false).
			Pluck("path", &paths).Error
		return paths, err
	}
	var paths []string
	err := s.DB.Table("sys_menus").
		Select("DISTINCT sys_menus.path").
		Joins("JOIN sys_role_menus ON sys_role_menus.sys_menu_id = sys_menus.id").
		Joins("JOIN sys_user_roles ON sys_user_roles.sys_role_id = sys_role_menus.sys_role_id").
		Joins("JOIN sys_roles ON sys_roles.id = sys_user_roles.sys_role_id").
		Where("sys_user_roles.sys_user_id = ? AND sys_roles.status = ? AND sys_menus.status = ? AND sys_menus.menu_type = ? AND sys_menus.path <> '' AND sys_menus.hidden = ?",
			userID, "active", "active", "C", false).
		Pluck("sys_menus.path", &paths).Error
	return paths, err
}

func (s *Service) HasPermission(userID uint, authSource string, codes ...string) bool {
	if len(codes) == 0 {
		return true
	}
	if s.IsSuperAdmin(userID, authSource) {
		return true
	}
	perms, err := s.UserPermissions(userID, authSource)
	if err != nil {
		return false
	}
	set := map[string]struct{}{}
	for _, p := range perms {
		set[p] = struct{}{}
	}
	for _, c := range codes {
		if _, ok := set[c]; ok {
			return true
		}
	}
	return false
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func BuildMenuTree(flat []model.SysMenu) []model.SysMenu {
	byID := map[uint]*model.SysMenu{}
	for i := range flat {
		m := flat[i]
		m.Children = nil
		byID[m.ID] = &flat[i]
		flat[i].Children = nil
	}
	// re-copy into map with pointers to slice elements after reset
	items := make([]model.SysMenu, len(flat))
	copy(items, flat)
	byID = map[uint]*model.SysMenu{}
	for i := range items {
		byID[items[i].ID] = &items[i]
	}
	var roots []model.SysMenu
	for i := range items {
		m := &items[i]
		if m.ParentID == nil || *m.ParentID == 0 {
			roots = append(roots, *m)
			continue
		}
		if parent, ok := byID[*m.ParentID]; ok {
			parent.Children = append(parent.Children, *m)
		} else {
			roots = append(roots, *m)
		}
	}
	// rebuild children from byID for nested correctness
	var walk func(id uint) []model.SysMenu
	walk = func(id uint) []model.SysMenu {
		var kids []model.SysMenu
		for i := range items {
			if items[i].ParentID != nil && *items[i].ParentID == id {
				node := items[i]
				node.Children = walk(node.ID)
				kids = append(kids, node)
			}
		}
		return kids
	}
	roots = nil
	for i := range items {
		if items[i].ParentID == nil || *items[i].ParentID == 0 {
			node := items[i]
			node.Children = walk(node.ID)
			roots = append(roots, node)
		}
	}
	return roots
}

func EnsureRoleMenus(db *gorm.DB, roleID uint, menuIDs []uint) error {
	var role model.SysRole
	if err := db.First(&role, roleID).Error; err != nil {
		return err
	}
	var menus []model.SysMenu
	if len(menuIDs) > 0 {
		if err := db.Where("id IN ?", menuIDs).Find(&menus).Error; err != nil {
			return err
		}
	}
	return db.Model(&role).Association("Menus").Replace(menus)
}

func EnsureUserRoles(db *gorm.DB, userID uint, roleIDs []uint) error {
	var user model.SysUser
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	var roles []model.SysRole
	if len(roleIDs) > 0 {
		if err := db.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
	}
	return db.Model(&user).Association("Roles").Replace(roles)
}

func RoleMenuIDs(db *gorm.DB, roleID uint) ([]uint, error) {
	var ids []uint
	err := db.Table("sys_role_menus").Where("sys_role_id = ?", roleID).Pluck("sys_menu_id", &ids).Error
	return ids, err
}

func UserRoleIDs(db *gorm.DB, userID uint) ([]uint, error) {
	var ids []uint
	err := db.Table("sys_user_roles").Where("sys_user_id = ?", userID).Pluck("sys_role_id", &ids).Error
	return ids, err
}

func FindLocalUser(db *gorm.DB, username string) (*model.SysUser, error) {
	var u model.SysUser
	err := db.Where("username = ?", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func MustHash(password string) string {
	h, err := HashPassword(password)
	if err != nil {
		panic(fmt.Sprintf("hash password: %v", err))
	}
	return h
}
