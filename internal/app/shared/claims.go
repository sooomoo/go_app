package shared

import "goapp/pkg/core"

type AuthorizedClaims struct {
	UserId          int64         `json:"userId"`
	Platform        core.Platform `json:"platform"`
	UserAgent       string        `json:"userAgent"`
	UserAgentHashed string        `json:"userAgentHashed"`
	ClientId        string        `json:"clientId"`
	Ip              string        `json:"ip"`
}
