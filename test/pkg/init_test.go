package pkg_test

import (
	"fmt"
	"goapp/pkg/ids"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()

	ids.IDSetNodeID(1)
	fmt.Println("init")
}
