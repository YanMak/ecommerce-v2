package inventory

import (
	"context"
	"errors"
	"time"

	invpb "github.com/YanMak/ecommerce/v2/gen/inventory/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ===== Доменные ошибки-заглушки (заменишь на свои из domain/app) =====
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnavailable  = errors.New("inventory unavailable")
	ErrDeadline     = errors.New("inventory deadline exceeded")
	ErrInternal     = errors.New("inventory internal error")
)

// ===== DTO-заглушки (заменишь на свои доменные) =====
type Stock struct {
	ItemID    int64
	Available int64
	// Locations можно добавить при надобности
	UpdatedAt time.Time
}

// ===== Клиент =====
type Client struct {
	cli     invpb.StockServiceClient
	timeout time.Duration
	retries int
}

// NewFromConn — простой конструктор.
func NewFromConn(conn *grpc.ClientConn, timeout time.Duration, retries int) *Client {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if retries < 0 {
		retries = 0
	}
	return &Client{
		cli:     invpb.NewStockServiceClient(conn),
		timeout: timeout,
		retries: retries,
	}
}

// New — если хочешь передать уже собранный StockServiceClient (например, обёрнутый интерсепторами).
func New(cli invpb.StockServiceClient, timeout time.Duration, retries int) *Client {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if retries < 0 {
		retries = 0
	}
	return &Client{cli: cli, timeout: timeout, retries: retries}
}

// GetStock — чтение одного товара (опц. по локации).
func (c *Client) GetStock(ctx context.Context, itemID int64, locationCode string) (Stock, error) {
	req := &invpb.GetStockRequest{ItemId: itemID, LocationCode: locationCode}
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		ctxT, cancel := context.WithTimeout(ctx, c.timeout)
		resp, err := c.cli.GetStock(ctxT, req)
		cancel()
		if err == nil {
			return toDTO(resp.GetStock()), nil
		}
		// Маппим коды и решаем — ретраить или нет
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			return Stock{}, ErrInvalidInput
		case codes.NotFound:
			return Stock{}, ErrNotFound
		case codes.DeadlineExceeded:
			lastErr = ErrDeadline
		case codes.Unavailable:
			lastErr = ErrUnavailable
		default:
			lastErr = ErrInternal
		}
		// Ретраим только на сетевые/временные
		if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			continue
		}
		return Stock{}, lastErr
	}
	return Stock{}, lastErr
}

// BatchGetStock — батч для листингов.
func (c *Client) BatchGetStock(ctx context.Context, itemIDs []int64, locationCode string) ([]Stock, error) {
	req := &invpb.BatchGetStockRequest{ItemIds: itemIDs, LocationCode: locationCode}
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		ctxT, cancel := context.WithTimeout(ctx, c.timeout)
		resp, err := c.cli.BatchGetStock(ctxT, req)
		cancel()
		if err == nil {
			out := make([]Stock, 0, len(resp.GetStocks()))
			for _, pb := range resp.GetStocks() {
				out = append(out, toDTO(pb))
			}
			return out, nil
		}
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.InvalidArgument:
			return nil, ErrInvalidInput
		case codes.NotFound:
			return nil, ErrNotFound
		case codes.DeadlineExceeded:
			lastErr = ErrDeadline
		case codes.Unavailable:
			lastErr = ErrUnavailable
		default:
			lastErr = ErrInternal
		}
		if st.Code() == codes.Unavailable || st.Code() == codes.DeadlineExceeded {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
			continue
		}
		return nil, lastErr
	}
	return nil, lastErr
}

// ===== Вспомогательное =====
func toDTO(pb *invpb.Stock) Stock {
	if pb == nil {
		return Stock{}
	}
	t := time.Time{}
	if ts := pb.GetUpdatedAt(); ts != nil {
		t = time.Unix(ts.GetSeconds(), int64(ts.GetNanos())).UTC()
	}
	return Stock{
		ItemID:    pb.GetItemId(),
		Available: pb.GetAvailable(),
		UpdatedAt: t,
	}
}
