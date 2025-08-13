package claims

import (
	"goapp/pkg/core"
	"goapp/pkg/ids"

	"github.com/gin-gonic/gin"
)

type AuthorizedClaims struct {
	UserId          ids.UID       `json:"userId"`
	Platform        core.Platform `json:"platform"`
	UserAgent       string        `json:"userAgent"`
	UserAgentHashed string        `json:"userAgentHashed"`
	ClientId        string        `json:"clientId"`
	Ip              string        `json:"ip"`
}

const (
	KeyClaims = "claims"
)

func GetClaims(c *gin.Context) *AuthorizedClaims {
	val, exist := c.Get(KeyClaims)
	if !exist {
		return nil
	}
	claims, ok := val.(*AuthorizedClaims)
	if !ok {
		return nil
	}
	return claims
}

func SaveClaims(ctx *gin.Context, claims *AuthorizedClaims) {
	if claims == nil {
		return
	}
	ctx.Set(KeyClaims, claims)
}
