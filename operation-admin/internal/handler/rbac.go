package handler

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/QuantumNous/new-api/operation-admin/internal/rbac"
	"github.com/gin-gonic/gin"
)

func (a *API) ListMenus(c *gin.Context) {
	var rows []model.SysMenu
	q := a.DB.Order("sort asc, id asc")
	if sub := c.Query("subsystem"); sub != "" {
		q = q.Where("subsystem = ?", sub)
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rbac.BuildMenuTree(rows)})
}

func (a *API) CreateMenu(c *gin.Context) {
	var m model.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if m.Status == "" {
		m.Status = "active"
	}
	if m.MenuType == "" {
		m.MenuType = "C"
	}
	if err := a.DB.Create(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": m})
}

func (a *API) UpdateMenu(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var m model.SysMenu
	if err := a.DB.First(&m, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "菜单不存在"})
		return
	}
	var req model.SysMenu
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	m.ParentID = req.ParentID
	m.Name = req.Name
	m.Path = req.Path
	m.Permission = req.Permission
	m.Icon = req.Icon
	m.ParentIcon = req.ParentIcon
	m.Subsystem = req.Subsystem
	m.MenuType = req.MenuType
	m.MenuGroup = req.MenuGroup
	m.Sort = req.Sort
	m.Status = req.Status
	m.Hidden = req.Hidden
	if err := a.DB.Save(&m).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": m})
}

func (a *API) DeleteMenu(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var child int64
	a.DB.Model(&model.SysMenu{}).Where("parent_id = ?", id).Count(&child)
	if child > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请先删除子菜单"})
		return
	}
	if err := a.DB.Delete(&model.SysMenu{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) ListRoles(c *gin.Context) {
	var rows []model.SysRole
	q := a.DB.Order("id asc")
	if name := c.Query("name"); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if err := q.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": rows})
}

func (a *API) GetRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var role model.SysRole
	if err := a.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "角色不存在"})
		return
	}
	menuIDs, _ := rbac.RoleMenuIDs(a.DB, role.ID)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"id": role.ID, "code": role.Code, "name": role.Name,
		"status": role.Status, "remark": role.Remark, "menu_ids": menuIDs,
	}})
}

func (a *API) CreateRole(c *gin.Context) {
	var req model.SysRole
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if err := a.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": req})
}

func (a *API) UpdateRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var role model.SysRole
	if err := a.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "角色不存在"})
		return
	}
	if role.Code == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "不可修改超级管理员角色编码"})
		return
	}
	var req struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	role.Name = req.Name
	role.Status = req.Status
	role.Remark = req.Remark
	if req.Code != "" {
		role.Code = req.Code
	}
	if err := a.DB.Save(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": role})
}

func (a *API) DeleteRole(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var role model.SysRole
	if err := a.DB.First(&role, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "角色不存在"})
		return
	}
	if role.Code == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "不可删除超级管理员角色"})
		return
	}
	_ = a.DB.Model(&role).Association("Menus").Clear()
	if err := a.DB.Delete(&role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) AssignRoleMenus(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		MenuIDs []uint `json:"menu_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	if err := rbac.EnsureRoleMenus(a.DB, uint(id), req.MenuIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	q := a.DB.Model(&model.SysUser{})
	if username := c.Query("username"); username != "" {
		q = q.Where("username LIKE ?", "%"+username+"%")
	}
	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	q.Count(&total)
	var users []model.SysUser
	if err := q.Preload("Roles").Order("id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	records := make([]gin.H, 0, len(users))
	for _, u := range users {
		roleIDs := make([]uint, 0, len(u.Roles))
		roleNames := make([]string, 0, len(u.Roles))
		for _, r := range u.Roles {
			roleIDs = append(roleIDs, r.ID)
			roleNames = append(roleNames, r.Name)
		}
		records = append(records, gin.H{
			"id": u.ID, "username": u.Username, "real_name": u.RealName,
			"phone": u.Phone, "email": u.Email, "status": u.Status,
			"role_ids": roleIDs, "role_names": roleNames,
			"created_at": u.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"records": records, "total": total, "page": page, "page_size": pageSize,
	}})
}

func (a *API) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		RealName string `json:"real_name"`
		Phone    string `json:"phone"`
		Email    string `json:"email"`
		Status   string `json:"status"`
		RoleIDs  []uint `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "用户名和密码必填"})
		return
	}
	hash, err := rbac.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if req.Status == "" {
		req.Status = "active"
	}
	u := model.SysUser{
		Username: req.Username, PasswordHash: hash, RealName: req.RealName,
		Phone: req.Phone, Email: req.Email, Status: req.Status,
	}
	if err := a.DB.Create(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	_ = rbac.EnsureUserRoles(a.DB, u.ID, req.RoleIDs)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": u.ID}})
}

func (a *API) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var u model.SysUser
	if err := a.DB.First(&u, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	var req struct {
		Password *string `json:"password"`
		RealName string  `json:"real_name"`
		Phone    string  `json:"phone"`
		Email    string  `json:"email"`
		Status   string  `json:"status"`
		RoleIDs  []uint  `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "参数错误"})
		return
	}
	u.RealName = req.RealName
	u.Phone = req.Phone
	u.Email = req.Email
	if req.Status != "" {
		u.Status = req.Status
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := rbac.HashPassword(*req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
			return
		}
		u.PasswordHash = hash
	}
	if err := a.DB.Save(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	if req.RoleIDs != nil {
		_ = rbac.EnsureUserRoles(a.DB, u.ID, req.RoleIDs)
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (a *API) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var u model.SysUser
	if err := a.DB.First(&u, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	if u.Username == "admin" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "不可删除内置管理员"})
		return
	}
	_ = a.DB.Model(&u).Association("Roles").Clear()
	if err := a.DB.Delete(&u).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
