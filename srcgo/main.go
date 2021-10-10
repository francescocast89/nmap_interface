package main

import (
	"fmt"
	"sync"
)

type A struct {
	Name string
}

func main() {
	//https://developer20.com/using-sync-pool/
	// done := make(chan bool)
	// if hostsList, err := NetworkScan("192.168.73.0/24"); err != nil {
	// 	fmt.Println(err)
	// } else {
	// 	fmt.Println(hostsList)
	// }
	// if err := PortScan("192.168.73.127"); err != nil {
	// 	fmt.Println(err)
	// }
	// networksList := []string{"192.168.3.1", "192.168.3.2", "192.168.3.3"}
	// ctx := context.Background()
	pool := &sync.Pool{
		New: func() interface{} {
			fmt.Println("returning new a")
			return nil
		},
	}
	one := pool.Get().(*A)
	one.Name = "first"
	fmt.Printf("one.Name = %s\n", one.Name)
	pool.Put(one)

	// wp := NewWorkerPool(1, len(networksList), ctx)
	// //go wp.StartCollector(ctx)
	// go func() {
	// 	for _, i := range networksList {
	// 		wp.jobs <- Job{
	// 			Id: JobID(fmt.Sprintf("Network Scan %s", i)),
	// 			ExecFn: func(ctx context.Context) (interface{}, error) {
	// 				return NetworkScan(i, ctx)
	// 			},
	// 		}
	// 	}
	// 	close(wp.jobs)
	// }()

	// wp.Run(ctx)

	// for i := range wp.results {
	// 	fmt.Println(i.Id, i.Value)
	// }

	// for i := 1; i <= 10; i++ {
	// 	wp.addJob(
	// 		fmt.Sprintf("%0.d", i),
	// 		NetworkScan("192.168.73.0/24", WithContext(ctx)),
	// 		// func(ctx context.Context) (interface{}, error) {
	// 		// 	n := rand.Intn(10)
	// 		// 	select {
	// 		// 	case <-time.After(time.Duration(n) * time.Second):
	// 		// 		return "hello", nil
	// 		// 	case <-ctx.Done():
	// 		// 		return nil, nil
	// 		// 	}
	// 		// },
	// 	)
	// }

	// for {
	// 	select {
	// 	case <-done:
	// 		return
	// 	}
	// }
}
