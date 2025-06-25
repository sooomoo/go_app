package services_test

import (
	"fmt"
	"goapp/internal/pkg/services"
	"goapp/pkg/collection"
	"goapp/pkg/core"
	"strconv"
	"sync"
	"testing"
)

func TestIdService(t *testing.T) {
	idSvc := services.NewDefaultIDService(1)
	userId := idSvc.NewUserID()
	orderId := idSvc.NewOrderID()
	uidv7 := idSvc.NewUUIDv7()
	cusRadix := core.NewCustomRadix34()
	fmt.Println(userId, cusRadix.Encode(int(userId)), strconv.FormatInt(userId, 36))
	fmt.Println(orderId, cusRadix.Encode(int(orderId)), strconv.FormatInt(orderId, 36))
	fmt.Println(uidv7)
	fmt.Println(core.NewUUIDv8())
	fmt.Println("objectid")
	fmt.Println("507f1f77bcf86cd79943901")

	userIdMP := collection.Set[string]{}
	wg := sync.WaitGroup{}
	wg.Add(100)
	for range 100 {
		go func() {
			defer wg.Done()
			for range 10000 {
				// id := idSvc.NextUserID()
				id := core.NewUUIDv8().String()
				userIdMP.Add(id)
			}
		}()
	}
	wg.Wait()

	fmt.Println("id count", userIdMP.Size())
}
