package pkg_test

import (
	"fmt"

	"github.com/google/uuid"
)

func init() {
	uuid.EnableRandPool()
	fmt.Println("init")
}
