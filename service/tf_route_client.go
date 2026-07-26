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

// ── 站点标识 ────────────────────────────────────────────────────

// siteKey 返回当前站点的唯一标识（从环境变量读取）。
func siteKey() string {
	return common.TokenFactorySiteKey()
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
		logger.LogError(context.Background(), "TFRouteClient init failed: "+err.Error())
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
		SiteKey:    siteKey(),
	}

	// Attach JWT to gRPC metadata.
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.SelectChannel(ctx, req)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("TFRouteService.SelectChannel failed: %v", err))
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
		SiteKey:       siteKey(),
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

// ── 用户级路由策略 gRPC 客户端方法 ──────────────────────────────

// UserRoutePolicy 用户路由策略完整视图（映射自 gRPC 响应）。
type UserRoutePolicy struct {
	Mode          string             `json:"mode"`
	GlobalMode    string             `json:"global_mode"`
	Groups        []UserModelGroup   `json:"groups"`
	UserOverrides []UserOverrideItem `json:"user_overrides"`
	GlobalOverrides []UserOverrideItem `json:"global_overrides"`
}

// UserModelGroup 用户视图中的模型分组。
type UserModelGroup struct {
	GroupKey     string             `json:"group_key"`
	DisplayName  string             `json:"display_name"`
	Models       []string           `json:"models"`
	ChannelCount int                `json:"channel_count"`
	Channels     []UserGroupChannel `json:"channels"`
}

// UserGroupChannel 用户视图中的渠道信息。
type UserGroupChannel struct {
	ChannelID        int      `json:"channel_id"`
	RouteSlug        string   `json:"route_slug"`
	Name             string   `json:"name"`
	MaskedName       string   `json:"masked_name"`
	ProviderSlug     string   `json:"provider_slug"`
	SupplierAlias    string   `json:"supplier_alias"`
	Status           int      `json:"status"`
	ModelsInGroup    []string `json:"models_in_group"`
	UserWeight       int      `json:"user_weight"`
	UserEnabled      bool     `json:"user_enabled"`
	UserConfigured   bool     `json:"user_configured"`
	GlobalWeight     int      `json:"global_weight"`
	GlobalEnabled    bool     `json:"global_enabled"`
	GlobalConfigured bool     `json:"global_configured"`
	Price            float64  `json:"price"`
}

// UserOverrideItem 归类覆盖项。
type UserOverrideItem struct {
	ID       uint   `json:"id"`
	RawModel string `json:"raw_model"`
	GroupKey string `json:"group_key"`
	IsUser   bool   `json:"is_user"`
}

// GetUserRoutePolicyFromTF 调用 TokenFactory 获取用户路由策略完整视图。
func GetUserRoutePolicyFromTF(jwtToken string, userID int, userRole int) (*UserRoutePolicy, error) {
	if !common.TokenFactoryRouteEnabled() {
		return nil, fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return nil, fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.GetUserRoutePolicy(ctx, &pb.GetUserRoutePolicyRequest{
		UserId:   int32(userID),
		UserRole: int32(userRole),
		SiteKey:  siteKey(),
	})
	if err != nil {
		return nil, fmt.Errorf("GetUserRoutePolicy gRPC failed: %w", err)
	}

	policy := &UserRoutePolicy{
		Mode:       resp.Mode,
		GlobalMode: resp.GlobalMode,
	}

	for _, g := range resp.Groups {
		group := UserModelGroup{
			GroupKey:     g.GroupKey,
			DisplayName:  g.DisplayName,
			Models:       g.Models,
			ChannelCount: int(g.ChannelCount),
		}
		for _, ch := range g.Channels {
			group.Channels = append(group.Channels, UserGroupChannel{
				ChannelID:        int(ch.ChannelId),
				RouteSlug:        ch.RouteSlug,
				Name:             ch.ChannelName,
				MaskedName:       ch.MaskedName,
				ProviderSlug:     ch.ProviderSlug,
				SupplierAlias:    ch.SupplierAlias,
				Status:           int(ch.Status),
				ModelsInGroup:    ch.ModelsInGroup,
				UserWeight:       int(ch.UserWeight),
				UserEnabled:      ch.UserEnabled,
				UserConfigured:   ch.UserConfigured,
				GlobalWeight:     int(ch.GlobalWeight),
				GlobalEnabled:    ch.GlobalEnabled,
				GlobalConfigured: ch.GlobalConfigured,
				Price:            ch.Price,
			})
		}
		policy.Groups = append(policy.Groups, group)
	}

	for _, o := range resp.UserOverrides {
		policy.UserOverrides = append(policy.UserOverrides, UserOverrideItem{
			ID: uint(o.Id), RawModel: o.RawModel, GroupKey: o.GroupKey, IsUser: o.IsUser,
		})
	}
	for _, o := range resp.GlobalOverrides {
		policy.GlobalOverrides = append(policy.GlobalOverrides, UserOverrideItem{
			ID: uint(o.Id), RawModel: o.RawModel, GroupKey: o.GroupKey, IsUser: o.IsUser,
		})
	}

	return policy, nil
}

// UpsertUserRouteModeToTF 调用 TokenFactory 更新用户路由模式。
func UpsertUserRouteModeToTF(jwtToken string, userID int, userRole int, mode string, resetMode bool) error {
	if !common.TokenFactoryRouteEnabled() {
		return fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	req := &pb.UpsertUserRoutePolicyRequest{
		UserId:    int32(userID),
		UserRole:  int32(userRole),
		ResetMode: resetMode,
		SiteKey:   siteKey(),
	}
	if !resetMode {
		req.Mode = &mode
	}

	resp, err := client.client.UpsertUserRoutePolicy(ctx, req)
	if err != nil {
		return fmt.Errorf("UpsertUserRoutePolicy gRPC failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("UpsertUserRoutePolicy failed: %s", resp.Error)
	}
	return nil
}

// UpsertUserWeightToTF 调用 TokenFactory 创建/更新用户归类权重。
func UpsertUserWeightToTF(jwtToken string, userID int, userRole int, groupKey string, channelID int, weight int, enabled bool) error {
	if !common.TokenFactoryRouteEnabled() {
		return fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.UpsertUserRoutePolicy(ctx, &pb.UpsertUserRoutePolicyRequest{
		UserId:   int32(userID),
		UserRole: int32(userRole),
		SiteKey:  siteKey(),
		Weight: &pb.UpsertWeightItem{
			GroupKey:  groupKey,
			ChannelId: int32(channelID),
			Weight:    int32(weight),
			Enabled:   enabled,
		},
	})
	if err != nil {
		return fmt.Errorf("UpsertUserWeight gRPC failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("UpsertUserWeight failed: %s", resp.Error)
	}
	return nil
}

// UpsertUserOverrideToTF 调用 TokenFactory 创建/更新用户模型归类覆盖。
func UpsertUserOverrideToTF(jwtToken string, userID int, userRole int, rawModel string, groupKey string) error {
	if !common.TokenFactoryRouteEnabled() {
		return fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.UpsertUserRoutePolicy(ctx, &pb.UpsertUserRoutePolicyRequest{
		UserId:   int32(userID),
		UserRole: int32(userRole),
		SiteKey:  siteKey(),
		Override: &pb.UpsertOverrideItem{
			RawModel: rawModel,
			GroupKey: groupKey,
		},
	})
	if err != nil {
		return fmt.Errorf("UpsertUserOverride gRPC failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("UpsertUserOverride failed: %s", resp.Error)
	}
	return nil
}

// DeleteUserWeightFromTF 调用 TokenFactory 删除用户归类权重。
func DeleteUserWeightFromTF(jwtToken string, userID int, userRole int, weightID uint32) error {
	if !common.TokenFactoryRouteEnabled() {
		return fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.DeleteUserModelGroupWeight(ctx, &pb.DeleteUserModelGroupWeightRequest{
		UserId:   int32(userID),
		UserRole: int32(userRole),
		WeightId: weightID,
		SiteKey:  siteKey(),
	})
	if err != nil {
		return fmt.Errorf("DeleteUserModelGroupWeight gRPC failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("DeleteUserModelGroupWeight failed: %s", resp.Error)
	}
	return nil
}

// DeleteUserOverrideFromTF 调用 TokenFactory 删除用户模型归类覆盖。
func DeleteUserOverrideFromTF(jwtToken string, userID int, userRole int, overrideID uint32) error {
	if !common.TokenFactoryRouteEnabled() {
		return fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.DeleteUserModelGroupOverride(ctx, &pb.DeleteUserModelGroupOverrideRequest{
		UserId:     int32(userID),
		UserRole:   int32(userRole),
		OverrideId: overrideID,
		SiteKey:    siteKey(),
	})
	if err != nil {
		return fmt.Errorf("DeleteUserModelGroupOverride gRPC failed: %w", err)
	}
	if !resp.Success {
		return fmt.Errorf("DeleteUserModelGroupOverride failed: %s", resp.Error)
	}
	return nil
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

// ── 用户同步 ────────────────────────────────────────────────────

// SyncUsersToTF 推送用户快照到 TokenFactory（按站点隔离）。
func SyncUsersToTF(jwtToken string, users []*pb.ExternalUserInfo) (int, error) {
	if !common.TokenFactoryRouteEnabled() {
		return 0, fmt.Errorf("TokenFactory route not enabled")
	}

	client, err := GetTFRouteClient()
	if err != nil {
		return 0, fmt.Errorf("TFRouteClient init failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ctx = appendGRPCAuth(ctx, jwtToken)

	resp, err := client.client.SyncUsers(ctx, &pb.SyncUsersRequest{
		SiteKey: siteKey(),
		Users:   users,
	})
	if err != nil {
		return 0, fmt.Errorf("SyncUsers gRPC failed: %w", err)
	}

	return int(resp.SyncedCount), nil
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
