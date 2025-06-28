package headers

import (
	"errors"
	"goapp/internal/app"
	"goapp/internal/app/services/crypto"
	"goapp/internal/app/stores"
	"goapp/pkg/core"
	"goapp/pkg/cryptos"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CookieKeyPlatform     string = "pla"
	CookieKeySessionId    string = "sid"
	CookieKeyAccessToken  string = "acc"
	CookieKeyRefreshToken string = "ref"
	CookieKeyClientId     string = "cli"
	CookieKeyCsrfToken    string = "csrf"
)

const (
	HeaderAuthorization string = "Authorization"
	HeaderUserAgent     string = "User-Agent"
	HeaderContentLength string = "Content-Length"
	HeaderContentType   string = "Content-Type"
	HeaderNonce         string = "X-Nonce"
	HeaderTimestamp     string = "X-Timestamp"
	HeaderPlatform      string = "X-Platform"
	HeaderSignature     string = "X-Signature"
	HeaderSession       string = "X-Session"
	HeaderRawType       string = "X-RawType"
	HeaderClientId      string = "X-Client"
	HeaderCsrfToken     string = "X-CSRF"
)

func GetTrimmedHeader(ctx *gin.Context, name string) string {
	return strings.TrimSpace(ctx.GetHeader(name))
}

func GetPlatform(ctx *gin.Context) core.Platform {
	platform := GetTrimmedHeader(ctx, HeaderPlatform)
	pla := core.PlatformFromString(platform)
	if pla == core.Unspecify {
		str, _ := ctx.Cookie(CookieKeyPlatform)
		if len(str) > 0 {
			pla = core.PlatformFromString(str)
		}
	}

	return pla
}

func GetAccessToken(ctx *gin.Context) string {
	pla := GetPlatform(ctx)
	// web单独处理
	if pla == core.Web {
		token, _ := ctx.Cookie(CookieKeyAccessToken)
		if len(token) > 0 {
			return token
		}
	}

	// 从请求头中获取令牌
	authHeader := ctx.GetHeader(HeaderAuthorization)
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(authHeader[7:])
}

func GetRefreshToken(ctx *gin.Context) string {
	// web单独处理
	if GetPlatform(ctx) == core.Web {
		token, _ := ctx.Cookie(CookieKeyRefreshToken)
		if len(token) > 0 {
			return token
		}
	}

	// 非 web 请求的 refresh-token 应该在请求体中加密传输
	return ""
}

func GetClientId(ctx *gin.Context) string {
	// web单独处理
	if GetPlatform(ctx) == core.Web {
		token, _ := ctx.Cookie(CookieKeyClientId)
		if len(token) > 0 {
			return token
		}
	}

	// 从请求头中获取ClientId
	return strings.TrimSpace(ctx.GetHeader(HeaderClientId))
}

func GetSessionId(ctx *gin.Context) string {
	sid := GetTrimmedHeader(ctx, HeaderSession)
	if len(sid) == 0 {
		sid, _ = ctx.Cookie(CookieKeySessionId)
	}

	return sid
}

func GetUserAgent(ctx *gin.Context) string {
	ua := ctx.Request.Header.Get(HeaderUserAgent)
	return strings.TrimSpace(ua)
}

func GetUserAgentHashed(ctx *gin.Context) string {
	ua := GetUserAgent(ctx)
	if len(ua) > 0 {
		return cryptos.HashSha256(ua)
	}
	return ""
}

func GetCsrfToken(ctx *gin.Context) string {
	csrf, _ := ctx.Cookie(CookieKeyCsrfToken)
	if len(csrf) > 0 {
		return csrf
	}
	return ctx.GetHeader(HeaderCsrfToken)
}

const (
	KeyClaims     = "claims"
	KeyClientKeys = "client_keys"
	KeyExtendData = "extend_data"
)

type RequestExtendData struct {
	Nonce     string
	Timestamp string
	Platform  core.Platform
	Signature string
	SessionId string
}

func GetExtendData(ctx *gin.Context) *RequestExtendData {
	val, exist := ctx.Get(KeyExtendData)
	if !exist {
		return nil
	}
	extendData, ok := val.(*RequestExtendData)
	if !ok {
		return nil
	}
	return extendData
}

func SaveExtendData(ctx *gin.Context) {
	platform := GetPlatform(ctx)
	sessionId := GetSessionId(ctx)
	nonce := GetTrimmedHeader(ctx, HeaderNonce)
	timestamp := GetTrimmedHeader(ctx, HeaderTimestamp)
	signature := GetTrimmedHeader(ctx, HeaderSignature)
	if len(sessionId) == 0 {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return
	}
	extendData := &RequestExtendData{
		Nonce:     nonce,
		Timestamp: timestamp,
		Platform:  platform,
		Signature: signature,
		SessionId: sessionId,
	}
	ctx.Set(KeyExtendData, extendData)
}

type SessionClientKeys struct {
	SignPubKey []byte // 客户端签名公钥
	BoxPubKey  []byte // 客户端加密公钥
	ShareKey   []byte // 与BoxPubKey协商出来的加密密钥
}

func SaveClientKeys(ctx *gin.Context) *SessionClientKeys {
	sessionId := GetSessionId(ctx)
	if len(sessionId) == 0 {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return nil
	}
	raw, err := crypto.Base64Decode(sessionId)
	if err != nil {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return nil
	}

	if len(raw) != 88 {
		ctx.AbortWithError(400, errors.New("bad session id"))
		return nil
	}

	for i := 17; i < len(raw); i++ {
		elem := raw[i]
		raw[i] = elem ^ raw[i%17]
	}

	signPubKey := raw[24:56]
	boxPubKey := raw[56:]
	if app.GetGlobal().GetAuthConfig().EnableCrypto {
		shareKey, err := crypto.NegotiateShareKey(boxPubKey, app.GetGlobal().GetAuthConfig().BoxKeyPair.PrivateKey)
		if err != nil {
			ctx.AbortWithError(400, errors.New("negotiate fail"))
			return nil
		}

		skeys := &SessionClientKeys{
			SignPubKey: signPubKey,
			BoxPubKey:  boxPubKey,
			ShareKey:   shareKey,
		}
		ctx.Set(KeyClientKeys, skeys)
		return skeys
	} else {
		skeys := &SessionClientKeys{
			SignPubKey: signPubKey,
			BoxPubKey:  boxPubKey,
			ShareKey:   nil,
		}
		ctx.Set(KeyClientKeys, skeys)
		return skeys
	}
}

func GetClientKeys(ctx *gin.Context) *SessionClientKeys {
	val, exist := ctx.Get(KeyClientKeys)
	if !exist {
		return nil
	}
	keys, ok := val.(*SessionClientKeys)
	if !ok {
		return nil
	}
	return keys
}

func GetClaims(c *gin.Context) *stores.AuthorizedClaims {
	val, exist := c.Get(KeyClaims)
	if !exist {
		return nil
	}
	claims, ok := val.(*stores.AuthorizedClaims)
	if !ok {
		return nil
	}
	return claims
}

func SaveClaims(ctx *gin.Context, claims *stores.AuthorizedClaims) {
	if claims == nil {
		return
	}
	ctx.Set(KeyClaims, claims)
}
