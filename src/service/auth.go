package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"blog_api/src/config"
	"blog_api/src/model"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte
var credentialResolveOnce sync.Once
var expectedUsername string
var expectedPassword string

// InitJWTSecret 初始化 JWT 密钥
func InitJWTSecret() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// 如果环境变量未设置，生成一个随机密钥
		randomBytes := make([]byte, 32)
		rand.Read(randomBytes)
		secret = hex.EncodeToString(randomBytes)
	}
	jwtSecret = []byte(secret)
}

// AuthService 认证服务
type AuthService struct{}

// NewAuthService 创建认证服务实例
func NewAuthService() *AuthService {
	if len(jwtSecret) == 0 {
		InitJWTSecret()
	}
	resolveExpectedCredentials()
	return &AuthService{}
}

// ValidateCredentials 验证用户名和密码
func (s *AuthService) ValidateCredentials(username, password string) bool {
	resolveExpectedCredentials()

	return username == expectedUsername && password == expectedPassword
}

func resolveExpectedCredentials() {
	credentialResolveOnce.Do(func() {
		if cfg, err := config.Load(); err == nil && cfg != nil {
			expectedUsername = cfg.WebPanelUser
			expectedPassword = cfg.WebPanelPwd
		}
		if expectedUsername == "" {
			expectedUsername = os.Getenv("WEB_PANEL_USER")
		}
		if expectedPassword == "" {
			expectedPassword = os.Getenv("WEB_PANEL_PWD")
		}

		if expectedPassword == "" {
			expectedUsername = "admin_" + randomHex(3)
			expectedPassword = randomHex(12)
			log.Printf("[auth] 未检测到 WEB_PANEL_PWD，已为本次启动生成临时后台账号: username=%s password=%s", expectedUsername, expectedPassword)
			return
		}

		if expectedUsername == "" {
			expectedUsername = "admin"
		}
		log.Printf("[auth] 已读取到后台凭证（仅显示前3位）: username=%s*** password=%s***", prefix3(expectedUsername), prefix3(expectedPassword))
	})
}

func randomHex(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		// 极端情况下兜底，避免生成失败导致无法登录
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b)
}

func prefix3(s string) string {
	runes := []rune(s)
	if len(runes) <= 3 {
		return s
	}
	return string(runes[:3])
}

// GenerateJWT 生成 JWT token
func (s *AuthService) GenerateJWT(username string) (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour)

	claims := &model.JWTClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ValidateJWT 验证 JWT token
func (s *AuthService) ValidateJWT(tokenString string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*model.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
