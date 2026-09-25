package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/operation-admin/internal/model"
	"github.com/QuantumNous/new-api/operation-admin/internal/newapi"
	"github.com/QuantumNous/new-api/operation-admin/internal/rbac"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	ErrNotRoot     = errors.New("仅超级管理员可通过 new-api 账号登录")
	ErrBadCreds    = errors.New("用户名或密码错误")
	ErrUserDisabled = errors.New("账号已停用")
)

type Claims struct {
	UserID      uint   `json:"uid"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	SuperAdmin  bool   `json:"super_admin"`
	AuthSource  string `json:"auth_source"` // local | newapi
	jwt.RegisteredClaims
}

type Service struct {
	Secret string
	Expire time.Duration
	Client *newapi.Client
	DB     *gorm.DB
	RBAC   *rbac.Service
}

func (s *Service) LoginLocalOrNewAPI(username, password string) (token string, require2FA bool, flowToken string, claims *Claims, err error) {
	if u, e := rbac.FindLocalUser(s.DB, username); e == nil {
		if u.Status != "active" {
			return "", false, "", nil, ErrUserDisabled
		}
		if !rbac.CheckPassword(u.PasswordHash, password) {
			return "", false, "", nil, ErrBadCreds
		}
		token, claims, err = s.issueLocal(u)
		return token, false, "", claims, err
	} else if !errors.Is(e, gorm.ErrRecordNotFound) {
		return "", false, "", nil, e
	}

	// Fallback: new-api Root login (bootstrap / break-glass)
	res, err := s.Client.Login(username, password)
	if err != nil {
		return "", false, "", nil, ErrBadCreds
	}
	if res.Require2FA {
		return "", true, res.FlowToken, nil, nil
	}
	token, claims, err = s.issueFromNewAPILogin(res)
	return token, false, "", claims, err
}

func (s *Service) Login2FA(flowToken, code string) (string, *Claims, error) {
	res, err := s.Client.Login2FA(flowToken, code)
	if err != nil {
		return "", nil, err
	}
	return s.issueFromNewAPILogin(res)
}

func (s *Service) LoginWithPAT(pat string) (string, *Claims, error) {
	u, err := s.Client.GetSelf(pat)
	if err != nil {
		return "", nil, err
	}
	if u.Role < newapi.RoleRootUser {
		return "", nil, ErrNotRoot
	}
	return s.issueNewAPI(uint(u.ID), u.Username, u.DisplayName)
}

func (s *Service) issueLocal(u *model.SysUser) (string, *Claims, error) {
	super := s.RBAC.IsSuperAdmin(u.ID, "local")
	display := u.RealName
	if display == "" {
		display = u.Username
	}
	claims := &Claims{
		UserID:      u.ID,
		Username:    u.Username,
		DisplayName: display,
		SuperAdmin:  super,
		AuthSource:  "local",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.Expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("local:%d", u.ID),
		},
	}
	token, err := s.sign(claims)
	return token, claims, err
}

func (s *Service) issueFromNewAPILogin(res *newapi.LoginResult) (string, *Claims, error) {
	role := res.Role
	username := res.Username
	display := res.DisplayName
	uid := res.UserID
	if res.AccessToken != "" {
		if u, err := s.Client.GetSelf(res.AccessToken); err == nil {
			role = u.Role
			username = u.Username
			display = u.DisplayName
			uid = u.ID
		}
	}
	if role < newapi.RoleRootUser {
		return "", nil, ErrNotRoot
	}
	return s.issueNewAPI(uint(uid), username, display)
}

func (s *Service) issueNewAPI(uid uint, username, display string) (string, *Claims, error) {
	claims := &Claims{
		UserID:      uid,
		Username:    username,
		DisplayName: display,
		SuperAdmin:  true,
		AuthSource:  "newapi",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.Expire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("newapi:%d", uid),
		},
	}
	token, err := s.sign(claims)
	return token, claims, err
}

func (s *Service) sign(claims *Claims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(s.Secret))
}

func (s *Service) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}
