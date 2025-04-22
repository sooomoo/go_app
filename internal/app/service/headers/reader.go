package headers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sooomo/niu"
)

const (
	CookieKeyPlatform     string = "pla"
	CookieKeySessionId    string = "sid"
	CookieKeyAccessToken  string = "acc"
	CookieKeyRefreshToken string = "ref"
	CookieKeyClientId     string = "cli"
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
)

func GetTrimmedHeader(ctx *gin.Context, name string) string {
	return strings.TrimSpace(ctx.GetHeader(name))
}

func GetPlatform(ctx *gin.Context) niu.Platform {
	platform := GetTrimmedHeader(ctx, HeaderPlatform)
	pla := niu.ParsePlatform(platform)
	if pla == niu.Unspecify {
		str, _ := ctx.Cookie(CookieKeyPlatform)
		if len(str) > 0 {
			pla = niu.ParsePlatform(str)
		}
	}

	return pla
}

func GetAccessToken(ctx *gin.Context) string {
	pla := GetPlatform(ctx)
	// web单独处理
	if pla == niu.Web {
		token, _ := ctx.Cookie(CookieKeyAccessToken)
		return token
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
	if GetPlatform(ctx) == niu.Web {
		token, _ := ctx.Cookie(CookieKeyRefreshToken)
		return token
	}

	// 非 web 请求的 refresh-token 应该在请求体中加密传输
	return ""
}

func GetClientId(ctx *gin.Context) string {
	// web单独处理
	if GetPlatform(ctx) == niu.Web {
		token, _ := ctx.Cookie(CookieKeyClientId)
		return token
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

func GetUserAgentHashed(ctx *gin.Context) string {
	ua := ctx.Request.Header.Get(HeaderUserAgent)
	ua = strings.TrimSpace(ua)
	if len(ua) > 0 {
		ua = niu.HashSha256(ua)
	}
	return ua
}
