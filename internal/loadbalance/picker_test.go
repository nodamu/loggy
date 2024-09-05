package loadbalance_test

import (
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/resolver"
	"loggy/internal/loadbalance"
	"testing"
)

func TestPickerNoSubConnAvailable(t *testing.T) {
	picker := &loadbalance.Picker{}
	for _, method := range []string{
		"/log.vX.Log/Produce",
		"/log.vX.Log/Consume",
	} {
		info := balancer.PickInfo{FullMethodName: method}
		result, err := picker.Pick(info)
		require.Error(t, err)
		require.Equal(t, balancer.ErrNoSubConnAvailable, err)
		require.Nil(t, result.SubConn)
	}
}

func TestPickerProducesToLeader(t *testing.T) {
	picker, subConns := setupTest()
	info := balancer.PickInfo{FullMethodName: "/log.vX.Log/Produce"}
	for i := 0; i < 5; i++ {
		result, err := picker.Pick(info)
		require.NoError(t, err)
		require.Equal(t, subConns[0], result.SubConn)
	}
}

// subConn implements balancer.SubConn.
type subConn struct {
	addrs []resolver.Address
}

func (s *subConn) GetOrBuildProducer(
	builder balancer.ProducerBuilder,
) (p balancer.Producer, close func()) {
	//TODO implement me
	panic("implement me")
}
func (s *subConn) Shutdown() {}
func (s *subConn) UpdateAddresses(addresses []resolver.Address) {
	s.addrs = addresses
}

func (s *subConn) Connect() {}

func setupTest() (*loadbalance.Picker, []*subConn) {
	var subConns []*subConn
	buildInfo := base.PickerBuildInfo{
		ReadySCs: make(map[balancer.SubConn]base.SubConnInfo),
	}
	for i := 0; i < 3; i++ {
		sc := &subConn{}
		addr := resolver.Address{
			Attributes: attributes.New("is_leader", i == 0),
		}
		// 0th sub conn is the leader
		sc.UpdateAddresses([]resolver.Address{addr})
		buildInfo.ReadySCs[sc] = base.SubConnInfo{Address: addr}
		subConns = append(subConns, sc)
	}
	picker := &loadbalance.Picker{}
	picker.Build(buildInfo)
	return picker, subConns
}
