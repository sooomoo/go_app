package pkg_test

import (
	"fmt"
	"goapp/pkg/ipregion"
	"testing"
)

func TestIp2Region(t *testing.T) {
	err := ipregion.Init("../../pkg/ipregion/ip2region.xdb")
	if err != nil {
		t.Error(err)
		return
	}

	result, err := ipregion.SearchFmt("60.255.166.251")
	if err != nil {
		t.Error(err)
	} else {
		fmt.Print(result)
	}
}
