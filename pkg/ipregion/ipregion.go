package ipregion

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
)

// 备注：并发使用，用整个 xdb 缓存创建的 searcher 对象可以安全用于并发。
var searcher *xdb.Searcher

// 初始化 Ip2Region
//
// dbPath: ip2region.xdb 文件的路径
func Init(dbPath string) error {
	// 1、从 dbPath 加载整个 xdb 到内存
	cBuff, err := xdb.LoadContentFromFile(dbPath)
	if err != nil {
		return err
	}

	// 2、用全局的 cBuff 创建完全基于内存的查询对象。
	searcher, err = xdb.NewWithBuffer(cBuff)
	if err != nil {
		return err
	}

	return nil
}

// 国家|区域|省份|城市|ISP，缺省的地域信息默认是0
func Search(ip string) (string, error) {
	return searcher.SearchByStr(ip)
}

// Ip2Region 搜索结果的格式化结构
type IPRegionFormated struct {
	Country  string `json:"country"`  // 国家
	Area     string `json:"area"`     // 区域
	Province string `json:"province"` // 省份
	City     string `json:"city"`     // 城市
	ISP      string `json:"isp"`      // ISP: 如四川广电
}

func (r IPRegionFormated) String() string {
	return fmt.Sprintf("{ coutry: %s, area: %s, province: %s, city: %s, isp: %s }", r.Country, r.Area, r.Province, r.City, r.ISP)
}

func getValue(val string) string {
	if val == "0" {
		return ""
	} else {
		return val
	}
}

// 解析搜索结果
func ParseSearchResult(result string) (*IPRegionFormated, error) {
	splits := strings.Split(result, "|")
	if len(splits) != 5 {
		return nil, errors.New("InvalidFormat")
	}
	out := &IPRegionFormated{}
	out.Country = getValue(splits[0])
	out.Area = getValue(splits[1])
	out.Province = getValue(splits[2])
	out.City = getValue(splits[3])
	out.ISP = getValue(splits[4])
	return out, nil
}

// 返回格式化之后的数据
func SearchFmt(ip string) (*IPRegionFormated, error) {
	result, err := searcher.SearchByStr(ip)
	if err != nil {
		return nil, err
	}
	return ParseSearchResult(result)
}

func Release() {
	inst := searcher
	searcher = nil
	if inst != nil {
		inst.Close()
	}
}
