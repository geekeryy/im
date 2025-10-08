package service

import (
	context "context"
	"im/pkg/config"
	"im/pkg/loadbalance"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type DiscoveryService struct {
	UnimplementedDiscoveryServer
	ctx          context.Context
	services     map[string][]*ServiceInfo
	mu           sync.RWMutex
	redisClient  *redis.Client
	redisKey     string
	ready        atomic.Bool
	initTimeout  time.Duration
	loadBalancer loadbalance.LoadBalancer
	logger       *slog.Logger
}

func NewDiscoveryService(ctx context.Context, logger *slog.Logger, conf *config.DiscoveryConfig) *DiscoveryService {
	serv := &DiscoveryService{
		ctx:         ctx,
		initTimeout: 10 * time.Second,
		redisKey:    "im:discovery:",
		services:    make(map[string][]*ServiceInfo),
		logger:      logger.With("service_uuid", uuid.New().String(), "service_name", "discovery"),
		redisClient: redis.NewClient(&redis.Options{
			Addr:     conf.RedisConfig.Addr,
			Password: conf.RedisConfig.Password,
			DB:       conf.RedisConfig.DB,
		}),
		loadBalancer: loadbalance.NewConsistentHashBalancer(),
	}

	serv.logger.Debug("discovery service config", "config", conf)

	if err := initServiceMap(serv); err != nil {
		serv.logger.Error("failed to init discovery service", "error", err)
	}
	serv.ready.Store(true)
	serv.logger.Info("discovery service ready")

	go func() {
		defer func() {
			if r := recover(); r != nil {
				serv.logger.Error("discovery service panic recover", "error", r)
			}
		}()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		select {
		case <-serv.ctx.Done():
			serv.logger.Info("discovery service context done")
			return
		case <-ticker.C:
			if err := initServiceMap(serv); err != nil {
				serv.logger.Error("failed to init discovery service", "error", err)
			}
		}
	}()

	return serv
}

func initServiceMap(s *DiscoveryService) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.initTimeout)
	defer cancel()
	serverList, err := s.redisClient.Keys(ctx, s.redisKey+"*").Result()
	if err != nil {
		s.logger.Error("failed to get service list " + s.redisKey + "*")
		return err
	}
	for _, server := range serverList {
		serviceList, err := s.getServiceRedis(ctx, server)
		if err != nil {
			return err
		}
		if err := s.saveServiceLocal(strings.TrimPrefix(server, s.redisKey), serviceList); err != nil {
			return err
		}
	}
	return nil
}

func (s *DiscoveryService) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	success, err := s.redisClient.SAdd(ctx, s.redisKey+req.ServiceName, req.ServiceAddress+":"+req.ServicePort).Result()
	if err != nil {
		return nil, err
	}
	if success == 0 {
		return nil, status.Errorf(codes.AlreadyExists, "service already exists")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[req.ServiceName] = append(s.services[req.ServiceName], &ServiceInfo{
		ServiceAddress: req.ServiceAddress,
		ServicePort:    req.ServicePort,
	})
	return nil, nil
}

func (s *DiscoveryService) Deregister(ctx context.Context, req *DeregisterRequest) (*DeregisterResponse, error) {
	success, err := s.redisClient.SRem(ctx, s.redisKey+req.ServiceName, req.ServiceAddress+":"+req.ServicePort).Result()
	if err != nil {
		return nil, err
	}
	if success == 0 {
		return nil, status.Errorf(codes.NotFound, "address not found in service")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, service := range s.services[req.ServiceName] {
		if service.ServiceAddress == req.ServiceAddress && service.ServicePort == req.ServicePort {
			s.services[req.ServiceName] = append(s.services[req.ServiceName][:i], s.services[req.ServiceName][i+1:]...)
			break
		}
	}
	return nil, nil
}

func (s *DiscoveryService) GetService(ctx context.Context, req *GetServiceRequest) (*GetServiceResponse, error) {
	if !s.ready.Load() {
		return nil, status.Errorf(codes.Unavailable, "service not ready")
	}
	service, err := s.getServiceLocal(req.ServiceName)
	if err != nil {
		service, err = s.getServiceRedis(ctx, req.ServiceName)
		if err != nil {
			return nil, err
		}
		if err := s.saveServiceLocal(req.ServiceName, service); err != nil {
			return nil, err
		}
	}
	return &GetServiceResponse{
		ServiceInfo: service,
	}, nil
}

func (s *DiscoveryService) GetServiceIP(ctx context.Context, req *GetServiceIPRequest) (*GetServiceIPResponse, error) {
	if !s.ready.Load() {
		return nil, status.Errorf(codes.Unavailable, "service not ready")
	}
	service, err := s.getServiceLocal(req.ServiceName)
	if err != nil {
		service, err = s.getServiceRedis(ctx, req.ServiceName)
		if err != nil {
			return nil, err
		}
		if err := s.saveServiceLocal(req.ServiceName, service); err != nil {
			return nil, err
		}
	}

	index := s.loadBalancer.Select(int64(len(service)), req.ClientKey)

	return &GetServiceIPResponse{
		ServiceAddress: service[index].ServiceAddress,
		ServicePort:    service[index].ServicePort,
	}, nil
}

func (s *DiscoveryService) Ready(ctx context.Context, req *ReadyRequest) (*ReadyResponse, error) {
	return &ReadyResponse{
		Ready: s.ready.Load(),
	}, nil
}

func (s *DiscoveryService) getServiceLocal(serviceName string) ([]*ServiceInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.services[serviceName]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "service not found in local")
	}
	if len(service) == 0 {
		return nil, status.Errorf(codes.NotFound, "service is null")
	}
	return service, nil
}

func (s *DiscoveryService) getServiceRedis(ctx context.Context, key string) ([]*ServiceInfo, error) {
	serviceList, err := s.redisClient.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	arr := make([]*ServiceInfo, 0, len(serviceList))
	for _, service := range serviceList {
		serviceInfoList := strings.SplitN(service, ":", 2)
		arr = append(arr, &ServiceInfo{
			ServiceAddress: serviceInfoList[0],
			ServicePort:    serviceInfoList[1],
		})
	}
	if len(arr) == 0 {
		return nil, status.Errorf(codes.NotFound, "service not found in redis")
	}
	return arr, nil
}

func (s *DiscoveryService) saveServiceLocal(serviceName string, service []*ServiceInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[serviceName] = service
	return nil
}
