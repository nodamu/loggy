package agent_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/travisjeffery/go-dynaport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	logv1 "loggy/api/v1"
	"loggy/internal/agent"
	"loggy/internal/config"
	"os"
	"testing"
	"time"
)

func TestAgent(t *testing.T) {
	serverTlSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		Server:        true,
		ServerAddress: "127.0.0.1",
	})
	require.NoError(t, err)
	peerTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.RootClientCertFile,
		KeyFile:       config.RootClientKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: "127.0.0.1",
		Server:        false,
	})
	require.NoError(t, err)
	var agents []*agent.Agent
	for i := 0; i < 3; i++ {
		ports := dynaport.Get(2)
		bindAddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
		rpcPort := ports[1]

		dataDir, err := os.MkdirTemp("", "agent-test-log")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, agents[0].Config.BindAddr)

		}
		agent, err := agent.New(agent.Config{
			Bootstrap:       i == 0,
			ServerTLSConfig: serverTlSConfig,
			PeerTLSConfig:   peerTLSConfig,
			DataDir:         dataDir,
			BindAddr:        bindAddr,
			RPCPort:         rpcPort,
			NodeName:        fmt.Sprintf("%d", i),
			StartJoinAddrs:  startJoinAddrs,
			ACLModelFile:    config.ACLModelFile,
			ACLPolicyFile:   config.ACLPolicyFile,
		})
		require.NoError(t, err)
		agents = append(agents, agent)
	}

	defer func() {
		for _, agent := range agents {
			err := agent.Shutdown()
			require.NoError(t, err)
			require.NoError(t, os.RemoveAll(agent.Config.DataDir))
		}
	}()
	time.Sleep(3 * time.Second)
	leaderClient := client(t, agents[0], peerTLSConfig)
	produceResponse, err := leaderClient.Produce(context.Background(), &logv1.ProduceRequest{
		Record: &logv1.Record{
			Value: []byte("foo"),
		},
	})
	require.NoError(t, err)
	consumeResponse, err := leaderClient.Consume(context.Background(), &logv1.ConsumeRequest{
		Offset: produceResponse.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	// wait until replication has finished
	time.Sleep(3 * time.Second)

	followerClient := client(t, agents[1], peerTLSConfig)
	consumeResponse, err = followerClient.Consume(context.Background(), &logv1.ConsumeRequest{
		Offset: produceResponse.Offset,
	})
	require.NoError(t, err)
	require.Equal(t, consumeResponse.Record.Value, []byte("foo"))

	consumeResponse, err = leaderClient.Consume(
		context.Background(),
		&logv1.ConsumeRequest{
			Offset: produceResponse.Offset + 1,
		})
	require.Error(t, err)
	require.Nil(t, consumeResponse)
	got := status.Code(err)
	want := status.Code(logv1.ErrOffsetOutOfRange{}.GRPCStatus().Err())
	require.Equal(t, got, want)
}

func client(t *testing.T, a *agent.Agent, tlsConfig *tls.Config) logv1.LogClient {
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
	rpcAddr, err := a.Config.RPCAddr()
	require.NoError(t, err)
	conn, err := grpc.NewClient(fmt.Sprintf("%s", rpcAddr), opts...)
	require.NoError(t, err)
	client := logv1.NewLogClient(conn)
	return client

}
