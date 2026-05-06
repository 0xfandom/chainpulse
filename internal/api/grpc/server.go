// Package grpcsrv implements the gRPC surface mirroring the REST API.
// Each method delegates to the same ReadStore + ReadCache used by the
// REST handlers — no duplicated query logic.
package grpcsrv

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/0xfandom/chainpulse/internal/api/grpc/pb"
	"github.com/0xfandom/chainpulse/internal/api/store"
	chainpulselog "github.com/0xfandom/chainpulse/internal/log"
)

// Server wraps a *grpc.Server bound to a chosen listener address.
type Server struct {
	pb.UnimplementedWalletServer
	pb.UnimplementedTokenServer
	pb.UnimplementedProtocolServer
	pb.UnimplementedChainServer

	store *store.ReadStore
	cache *store.ReadCache

	grpc *grpc.Server
}

// New constructs a gRPC server. The same store + cache are reused from
// the REST handler stack.
func New(st *store.ReadStore, cache *store.ReadCache) *Server {
	s := &Server{store: st, cache: cache, grpc: grpc.NewServer()}
	pb.RegisterWalletServer(s.grpc, s)
	pb.RegisterTokenServer(s.grpc, s)
	pb.RegisterProtocolServer(s.grpc, s)
	pb.RegisterChainServer(s.grpc, s)
	reflection.Register(s.grpc)
	return s
}

// Serve binds to addr and serves until the listener errors. Use Stop /
// GracefulStop to terminate.
func (s *Server) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("grpc listen %s: %w", addr, err)
	}
	l := chainpulselog.Component("api_grpc")
	l.Info().Str("addr", addr).Msg("listening")
	return s.grpc.Serve(lis)
}

// ServeListener is the test/in-process entry point.
func (s *Server) ServeListener(lis net.Listener) error {
	return s.grpc.Serve(lis)
}

// GracefulStop drains in-flight RPCs.
func (s *Server) GracefulStop() {
	if s == nil || s.grpc == nil {
		return
	}
	s.grpc.GracefulStop()
}

// validProtocols mirrors the REST whitelist.
var validProtocols = map[string]struct{}{
	"erc20": {}, "uniswap_v3": {}, "aave_v3": {}, "compound_v3": {},
}

// GetWalletPositions returns the wallet's DeFi positions.
func (s *Server) GetWalletPositions(ctx context.Context, req *pb.GetWalletPositionsRequest) (*pb.GetWalletPositionsResponse, error) {
	wallet := strings.ToLower(req.GetWallet())
	if wallet == "" {
		return nil, errors.New("wallet required")
	}
	rows, err := s.store.WalletDefiPositions(ctx, wallet, 200)
	if err != nil {
		return nil, err
	}
	out := &pb.GetWalletPositionsResponse{Positions: make([]*pb.PositionRow, 0, len(rows))}
	for _, r := range rows {
		if req.GetChainId() != 0 && r.ChainID != req.GetChainId() {
			continue
		}
		if req.GetProtocol() != "" && r.Protocol != req.GetProtocol() {
			continue
		}
		out.Positions = append(out.Positions, &pb.PositionRow{
			ChainId: r.ChainID, Protocol: r.Protocol, EventType: r.EventType,
			UserAddr: r.UserAddr, TokenA: r.TokenA, AmountA: r.AmountA,
			Params: r.Params, Timestamp: r.Timestamp.Unix(),
		})
	}
	return out, nil
}

// GetWalletBalances returns Redis-cached balances per token.
func (s *Server) GetWalletBalances(ctx context.Context, req *pb.GetWalletBalancesRequest) (*pb.GetWalletBalancesResponse, error) {
	wallet := strings.ToLower(req.GetWallet())
	if wallet == "" {
		return nil, errors.New("wallet required")
	}
	if req.GetChainId() == 0 {
		return nil, errors.New("chain_id required")
	}
	if s.cache == nil {
		return &pb.GetWalletBalancesResponse{ChainId: req.GetChainId(), Balances: map[string]string{}}, nil
	}
	balances, err := s.cache.GetBalances(ctx, wallet, req.GetChainId())
	if err != nil {
		return nil, err
	}
	return &pb.GetWalletBalancesResponse{ChainId: req.GetChainId(), Balances: balances}, nil
}

// GetWalletHistory returns recent decoded events for the wallet.
func (s *Server) GetWalletHistory(ctx context.Context, req *pb.GetWalletHistoryRequest) (*pb.GetWalletHistoryResponse, error) {
	wallet := strings.ToLower(req.GetWallet())
	if wallet == "" {
		return nil, errors.New("wallet required")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.store.WalletHistory(ctx, wallet, limit)
	if err != nil {
		return nil, err
	}
	out := &pb.GetWalletHistoryResponse{Events: make([]*pb.HistoryRow, 0, len(rows))}
	for _, r := range rows {
		out.Events = append(out.Events, &pb.HistoryRow{
			ChainId: r.ChainID, BlockNumber: r.BlockNumber, TxHash: r.TxHash, LogIndex: r.LogIndex,
			Protocol: r.Protocol, EventType: r.EventType, UserAddr: r.UserAddr,
			TokenA: r.TokenA, AmountA: r.AmountA, Params: r.Params,
			Timestamp: r.Timestamp.Unix(),
		})
	}
	return out, nil
}

// GetTokenTransfers returns recent transfers for a token contract.
func (s *Server) GetTokenTransfers(ctx context.Context, req *pb.GetTokenTransfersRequest) (*pb.GetTokenTransfersResponse, error) {
	token := strings.ToLower(req.GetToken())
	if token == "" {
		return nil, errors.New("token required")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	rows, err := s.store.TokenTransfers(ctx, token, limit)
	if err != nil {
		return nil, err
	}
	out := &pb.GetTokenTransfersResponse{Transfers: make([]*pb.TransferRow, 0, len(rows))}
	for _, r := range rows {
		out.Transfers = append(out.Transfers, &pb.TransferRow{
			ChainId: r.ChainID, BlockNumber: r.BlockNumber, TxHash: r.TxHash, LogIndex: r.LogIndex,
			Token: r.Token, FromAddr: r.FromAddr, ToAddr: r.ToAddr, Amount: r.Amount,
			Timestamp: r.Timestamp.Unix(),
		})
	}
	return out, nil
}

// GetProtocolStats returns 24h stats for a known protocol.
func (s *Server) GetProtocolStats(ctx context.Context, req *pb.GetProtocolStatsRequest) (*pb.GetProtocolStatsResponse, error) {
	if _, ok := validProtocols[req.GetProtocol()]; !ok {
		return nil, fmt.Errorf("unknown protocol %q", req.GetProtocol())
	}
	stats, err := s.store.ProtocolStats(ctx, req.GetProtocol())
	if err != nil {
		return nil, err
	}
	out := &pb.GetProtocolStatsResponse{Stats: make([]*pb.ProtocolStat, 0, len(stats))}
	for _, s := range stats {
		if req.GetChainId() != 0 && s.ChainID != req.GetChainId() {
			continue
		}
		out.Stats = append(out.Stats, &pb.ProtocolStat{
			ChainId: s.ChainID, Protocol: s.Protocol,
			EventCount: s.EventCount, UniqueUsers: s.UniqueUsers,
			LastSeen: s.LastSeen.Unix(),
		})
	}
	return out, nil
}

// GetChainBlocks returns recent indexed blocks for a chain.
func (s *Server) GetChainBlocks(ctx context.Context, req *pb.GetChainBlocksRequest) (*pb.GetChainBlocksResponse, error) {
	if req.GetChainId() == 0 {
		return nil, errors.New("chain_id required")
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.store.ChainRecentBlocks(ctx, req.GetChainId(), limit)
	if err != nil {
		return nil, err
	}
	out := &pb.GetChainBlocksResponse{Blocks: make([]*pb.BlockSummary, 0, len(rows))}
	for _, r := range rows {
		out.Blocks = append(out.Blocks, &pb.BlockSummary{
			ChainId: r.ChainID, BlockNumber: r.BlockNumber,
			EventCount: r.EventCount, LatestTs: r.LatestTS.Unix(),
		})
	}
	return out, nil
}

// dialTimeout keeps tests fast; not used by the runtime path.
var dialTimeout = 5 * time.Second
var _ = dialTimeout
