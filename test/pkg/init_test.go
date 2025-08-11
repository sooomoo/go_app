package pkg_test

import (
	"fmt"
	"goapp/pkg/core"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()

	core.IDSetNodeID(1)
	fmt.Println("init")
}
