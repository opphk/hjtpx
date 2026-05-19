package circuitbreaker

import (
	"context"
	"errors"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryClientInterceptor(manager *CircuitBreakerManager, serviceName string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		cb := manager.GetOrCreate(serviceName, nil)

		result, err := cb.Execute(ctx, func() (interface{}, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			return reply, err
		})

		if err != nil {
			log.Printf("[CircuitBreaker] gRPC call failed for %s: %v", method, err)
			
			if errors.Is(err, ErrCircuitOpen) {
				return status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
			}
			return err
		}

		if result != nil {
			*reply.(*interface{}) = result
		}
		return nil
	}
}

func UnaryServerInterceptor(manager *CircuitBreakerManager) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		serviceName := info.FullMethod

		cb := manager.GetOrCreate(serviceName, nil)

		result, err := cb.Execute(ctx, func() (interface{}, error) {
			return handler(ctx, req)
		})

		if err != nil {
			log.Printf("[CircuitBreaker] gRPC server handler failed for %s: %v", serviceName, err)
			
			if errors.Is(err, ErrCircuitOpen) {
				return nil, status.Error(codes.Unavailable, "service circuit breaker is open")
			}
			return nil, err
		}

		return result, nil
	}
}

func StreamClientInterceptor(manager *CircuitBreakerManager, serviceName string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		cb := manager.GetOrCreate(serviceName, nil)

		result, err := cb.Execute(ctx, func() (interface{}, error) {
			stream, err := streamer(ctx, desc, cc, method, opts...)
			return stream, err
		})

		if err != nil {
			log.Printf("[CircuitBreaker] gRPC stream call failed for %s: %v", method, err)
			
			if errors.Is(err, ErrCircuitOpen) {
				return nil, status.Error(codes.Unavailable, "service unavailable due to circuit breaker")
			}
			return nil, err
		}

		return result.(grpc.ClientStream), nil
	}
}

func StreamServerInterceptor(manager *CircuitBreakerManager) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		serviceName := info.FullMethod

		cb := manager.GetOrCreate(serviceName, nil)

		_, err := cb.Execute(ss.Context(), func() (interface{}, error) {
			return nil, handler(srv, ss)
		})

		if err != nil {
			log.Printf("[CircuitBreaker] gRPC stream handler failed for %s: %v", serviceName, err)
			
			if errors.Is(err, ErrCircuitOpen) {
				return status.Error(codes.Unavailable, "service circuit breaker is open")
			}
			return err
		}

		return nil
	}
}