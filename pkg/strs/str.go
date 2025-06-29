package strs

import (
	"regexp"
	"strconv"
	"unicode/utf8"
)

// 是否是固定长度的整数
func IsFixedLengthInt(value string, length int) bool {
	if utf8.RuneCountInString(value) != length {
		return false
	}

	_, err := strconv.Atoi(value)
	return err == nil
}

// 指定的字符串的长度是否符合要求
func IsLengthInRange(value string, min, max int) bool {
	l := utf8.RuneCountInString(value)
	return l >= min && l <= max
}

// 指定的字符串是否为空
func IsEmpty(value string) bool {
	return utf8.RuneCountInString(value) == 0
}

// 获取字符串的长度，此方法可以正确获取中文字符的个数，
// 与 len(value) 不同，len(value) 返回的是字节长度，而不是字符个数
func Length(value string) int {
	return utf8.RuneCountInString(value)
}

const (
	urlRegexStr   = `((http|ftp|https)://)(([a-zA-Z0-9\._-]+\.[a-zA-Z]{2,6})|([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}))(:[0-9]{1,4})*(/[a-zA-Z0-9\&%_\./-~-]*)?`
	emailRegexStr = `^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$`
)

var (
	countryCodeRegex = regexp.MustCompile(`^\d{3}$`)              // 国家代码正则表达式
	phoneRegex       = regexp.MustCompile(`^1[3-9]\d{9}$`)        // 手机号正则表达式
	telephoneRegex   = regexp.MustCompile(`^(\d{3,4}-)?\d{6,8}$`) // 座机号码
	urlRegex         = regexp.MustCompile(urlRegexStr)            // URL
	emailRegex       = regexp.MustCompile(emailRegexStr)          // Email Address
	colorRegex       = regexp.MustCompile(`^#[a-fA-F0-9]{8}$`)    // 颜色值
	numberRegex      = regexp.MustCompile(`^[0-9]*$`)             // 数字
)

// 是否是有效的手机号
func IsCellPhone(value string) bool {
	return phoneRegex.MatchString(value)
}

// 是否是有效的国家代码，如中国为 86
func IsCountryCode(value string) bool {
	return countryCodeRegex.MatchString(value)
}

// 是否是有效的座机号
func IsTelephone(value string) bool {
	return telephoneRegex.MatchString(value)
}

// 是否是链接地址
func IsUrl(value string, allowEmpty bool) bool {
	if allowEmpty && utf8.RuneCountInString(value) == 0 {
		return true
	}

	return urlRegex.MatchString(value)
}

// 是否是有效的邮件地址
func IsEmail(value string) bool {
	return emailRegex.MatchString(value)
}

// 是否是有效的颜色值
func IsColor(value string) bool {
	return colorRegex.MatchString(value)
}

// 是否是数字
func IsNumber(value string) bool {
	return numberRegex.MatchString(value)
}
