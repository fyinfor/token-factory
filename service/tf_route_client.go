package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	pb "github.com/QuantumNous/new-api/proto/route"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	tfRouteClient   *TFRouteClient
	tfRouteClientMu sync.RWMutex
)

// TFRouteClient wraps the gRPC client to TokenFactory RouteService.
type TFRouteClient struct {
	conn   *grpc.ClientConn
	client pb.RouteServiceClient
	addr   string
}

// GetTFRouteClient returns a singleton gRPC client (lazy init).
func GetTFRouteClient() (*TFRouteClient, error) {
	tfRouteClientMu.RLock()
	if tfRouteClient != nil {
		tfRouteClientMu.RUnlock()
		return tfRouteClient, nil
	}
	tfRouteClientMu.RUnlock()

	tfRouteClientMu.Lock()
	defer tfRouteClientMu.Unlock()

	if tfRouteClient != nil {
		return tfRouteClient, nil
	}

	addr := common.TokenFactoryGRPCEndpoint()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("gRPC dial %s: %w", addr, err)
	}
	tfRouteClient = &TFRouteClient{
		conn:   conn,
		client: pb.NewRouteServiceClient(conn),
		addr:   addr,
	}
	return tfRouteClient, nil
}

// SelectChannelFromTF calls TokenFactory to get ordered channel IDs.
// Returns (ordered_ids, strategy, group_key, fallback, error).
// group_key 为命中归类的 key（来自 SelectChannelResponse.policy_name），用于黏性缓存键。
// If TokenFactory is disabled or unavailable, returns nil slice with fallback=true.
func SelectChannelFromTF(jwtToken string, model string, group string, userID int, userRole int, candidates []*pb.ChannelCandidate) ([]int32, string, string, bool, error) {
	if !common.TokenFactoryRouteEnabled() {
		return nil, "", "", true, nil
	}

	client, err := GetTFRouteClient()
	if err != nil {
		logger.LogError(nil, "TFRouteClient init failed: "+err.Error())
		return nil, "", "", true, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &pb.SelectChannelRequest{
		Model:      model,
		Group:      group,
		UserId:     int32(userID),
		UserRole:   int32(userRole),
		Candidates: candidates,
	}

	// Attach JWT to gRPC metadata.
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.SelectChannel(ctx, req)
	if err != nil {
		logger.LogError(nil, fmt.Sprintf("TFRouteService.SelectChannel failed: %v", err))
		return nil, "", "", true, err
	}

	return resp.OrderedChannelIds, resp.Strategy, resp.PolicyName, resp.Fallback, nil
}

// SyncChannelsToTF pushes channel snapshots to TokenFactory.
func SyncChannelsToTF(jwtToken string, channels []*pb.ChannelSnapshot) (int, error) {
	client, err := GetTFRouteClient()
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	req := &pb.SyncChannelsRequest{
		Channels:      channels,
		SyncTimestamp: time.Now().Unix(),
	}

	resp, err := client.client.SyncChannels(ctx, req)
	if err != nil {
		return 0, err
	}

	return int(resp.SyncedCount), nil
}

// CloseTFRouteClient closes the gRPC connection.
func CloseTFRouteClient() {
	tfRouteClientMu.Lock()
	defer tfRouteClientMu.Unlock()
	if tfRouteClient != nil {
		_ = tfRouteClient.conn.Close()
		tfRouteClient = nil
	}
}

// ChannelRouteInfo holds channel fields needed for routing decisions.
type ChannelRouteInfo struct {
	ID           int
	Name         string
	Type         int
	Models       string
	Group        string
	Priority     int
	Weight       int
	Status       int
	ProviderSlug string
	// Price 该渠道在请求模型下的单价信号（相对值，越低越便宜；0=未知）。
	// 供 TokenFactory 价格优 / 价格相关策略排序使用。
	Price float64
}

// BuildChannelCandidates converts local channel info to gRPC candidates.
func BuildChannelCandidates(channels []ChannelRouteInfo) []*pb.ChannelCandidate {
	result := make([]*pb.ChannelCandidate, 0, len(channels))
	for _, ch := range channels {
		result = append(result, &pb.ChannelCandidate{
			ChannelId:    int32(ch.ID),
			ChannelName:  ch.Name,
			ChannelType:  int32(ch.Type),
			Models:       ch.Models,
			Group:        ch.Group,
			Priority:     int32(ch.Priority),
			Weight:       int32(ch.Weight),
			Status:       int32(ch.Status),
			ProviderSlug: ch.ProviderSlug,
			Price:        ch.Price,
			Healthy:      true,
		})
	}
	return result
}

// appendGRPCAuth adds JWT token to gRPC metadata.
func appendGRPCAuth(ctx context.Context, jwtToken string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+jwtToken)
}

// ParseModelsList splits comma-separated models string.
func ParseModelsList(models string) []string {
	if models == "" {
		return nil
	}
	parts := strings.Split(models, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
