package scrapers

import "strings"

type MineType struct {
	value string
}

func NewMineType(value string) *MineType {
	segs := strings.Split(value, ";")
	if len(segs) > 0 {
		value = strings.TrimSpace(segs[0])
	}
	return &MineType{value: value}
}

func (c MineType) GetValue() string {
	return c.value
}

func (c MineType) IsPdf() bool {
	return c.value == "application/pdf"
}

func (c MineType) IsHtml() bool {
	return strings.EqualFold(c.value, "text/html")
}

func (c MineType) IsPlainText() bool {
	return strings.EqualFold(c.value, "text/plain")
}

func (c MineType) IsJpeg() bool {
	return strings.EqualFold(c.value, "image/jpeg")
}

func (c MineType) IsPng() bool {
	return strings.EqualFold(c.value, "image/png")
}

func (c MineType) IsGif() bool {
	return strings.EqualFold(c.value, "image/gif")
}

func (c MineType) IsWebp() bool {
	return strings.EqualFold(c.value, "image/webp")
}

func (c MineType) IsBmp() bool {
	return strings.EqualFold(c.value, "image/bmp")
}

// PowerPoint 2007及以后版本基于XML的开放文档格式。这是目前最常用的类型
func (c MineType) IsPPTX() bool {
	return strings.EqualFold(c.value, "application/vnd.openxmlformats-officedocument.presentationml.presentation")
}

// Microsoft PowerPoint 97-2003创建的二进制格式演示文稿
func (c MineType) IsPPT2003() bool {
	return strings.EqualFold(c.value, "application/vnd.ms-powerpoint")
}
