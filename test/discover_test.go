package test

import (
	"context"
	"fmt"
	"im/server/discovery"
	"im/server/discovery/rpc/service"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestDiscover(t *testing.T) {
	go func() {
		os.Setenv("IM_DISCOVERY_ADDR", ":8085")
		discovery.Run()
	}()

	conn, err := grpc.NewClient("localhost:8085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	discoveryClient := service.NewDiscoveryClient(conn)

	t.Run("register", func(t *testing.T) {

		_, err = discoveryClient.Register(context.Background(), &service.RegisterRequest{
			ServiceName:    "test",
			ServiceAddress: "localhost",
			ServicePort:    "8085",
		})
		if err != nil {
			t.Fatalf("failed to register: %v", err)
		}
		_, err = discoveryClient.Register(context.Background(), &service.RegisterRequest{
			ServiceName:    "test",
			ServiceAddress: "localhost",
			ServicePort:    "8086",
		})
		if err != nil {
			t.Fatalf("failed to register: %v", err)
		}

	})

	t.Run("get service", func(t *testing.T) {
		service, err := discoveryClient.GetService(context.Background(), &service.GetServiceRequest{
			ServiceName: "test",
		})
		if err != nil {
			t.Fatalf("failed to get service: %v", err)
		}
		t.Logf("service: %v", service)

	})

	t.Run("get service ip", func(t *testing.T) {
		serviceIp, err := discoveryClient.GetServiceIP(context.Background(), &service.GetServiceIPRequest{
			ServiceName: "test",
			ClientKey:   "test",
		})
		if err != nil {
			t.Fatalf("failed to get service ip: %v", err)
		}
		t.Logf("service ip: %v", serviceIp)
	})

	t.Run("deregister", func(t *testing.T) {

		_, err = discoveryClient.Deregister(context.Background(), &service.DeregisterRequest{
			ServiceName:    "test",
			ServiceAddress: "localhost",
			ServicePort:    "8085",
		})
		if err != nil {
			t.Fatalf("failed to deregister: %v", err)
		}

		_, err = discoveryClient.Deregister(context.Background(), &service.DeregisterRequest{
			ServiceName:    "test",
			ServiceAddress: "localhost",
			ServicePort:    "8086",
		})
		if err != nil {
			t.Fatalf("failed to deregister: %v", err)
		}

		_, err = discoveryClient.Deregister(context.Background(), &service.DeregisterRequest{
			ServiceName:    "not exist",
			ServiceAddress: "localhost",
			ServicePort:    "8085",
		})
		if err == nil {
			t.Fatalf("should not deregister: %v", err)
		}
	})

}

func TestMultiDiscover(t *testing.T) {
	discoveryNum := 10
	clientNum := 100
	loopNum := 100
	ip := "localhost"
	port := 8085
	for i := 0; i < discoveryNum; i++ {
		os.Setenv("IM_DISCOVERY_ADDR", ip+":"+strconv.Itoa(port+i))
		go discovery.Run()
		time.Sleep(time.Millisecond * 100)
	}

	conn, err := grpc.NewClient(ip+":"+strconv.Itoa(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	discoveryClient := service.NewDiscoveryClient(conn)

	ctx := context.Background()
	discoveryClient.Ready(ctx, &service.ReadyRequest{})

	if ready, err := discoveryClient.Ready(ctx, &service.ReadyRequest{}); err != nil {
		t.Fatalf("failed to get ready: %v", err)
	} else {
		if !ready.Ready {
			t.Fatal("failed to get ready")
		}
	}

	discoveryClient.Register(ctx, &service.RegisterRequest{
		ServiceName:    "test",
		ServiceAddress: ip,
		ServicePort:    "8085",
	})
	discoveryClient.Register(ctx, &service.RegisterRequest{
		ServiceName:    "test",
		ServiceAddress: ip,
		ServicePort:    "8086",
	})

	var wg sync.WaitGroup
	ipMap := make(map[string]*atomic.Int64, 0)

	ipMap[ip+":8085"] = &atomic.Int64{}
	ipMap[ip+":8086"] = &atomic.Int64{}

	for i := 0; i < clientNum; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			conn, err := grpc.NewClient(ip+":"+strconv.Itoa(port+i%discoveryNum), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Errorf("failed to dial: %v", err)
			}
			defer conn.Close()

			discoveryClient := service.NewDiscoveryClient(conn)

			for j := 0; j < loopNum; j++ {
				serviceIp, err := discoveryClient.GetServiceIP(ctx, &service.GetServiceIPRequest{
					ServiceName: "test",
					ClientKey:   "test-" + strconv.Itoa(i),
				})
				if err != nil {
					t.Errorf("failed to get service ip: %v", err)
					continue // 跳过这次循环，避免空指针解引用
				}
				if serviceIp != nil && serviceIp.ServiceAddress != "" && serviceIp.ServicePort != "" {
					if _, ok := ipMap[fmt.Sprintf("%s:%s", serviceIp.ServiceAddress, serviceIp.ServicePort)]; ok {
						ipMap[fmt.Sprintf("%s:%s", serviceIp.ServiceAddress, serviceIp.ServicePort)].Add(1)
					}
				}
			}
		}(i)

	}
	wg.Wait()
	for ip, count := range ipMap {
		t.Logf("ip: %s, count: %d", ip, count)
	}

}
