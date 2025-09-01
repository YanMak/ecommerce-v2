package grpcstock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	invpb "github.com/YanMak/ecommerce/v2/gen/inventory/v1"
)

// ===== ПОРТ ПРИЛОЖЕНИЯ (use case интерфейс) =====

// InventoryQueries — входной (application) порт для чтения остатков.
type InventoryQueries interface {
	// Если locationCode == "", вернуть агрегат по всем локациям.
	// Иначе — можно вернуть все локации, а адаптер отфильтрует сам.
	GetStock(ctx context.Context, itemID int64, locationCode string) (StockDTO, error)

	// Возвращает stocks в произвольном порядке (адаптер сам упорядочит под запрос).
	BatchGetStock(ctx context.Context, itemIDs []int64, locationCode string) ([]StockDTO, error)
}

// Сентинелы/чекеры доменных ошибок (примерные)
var (
	ErrNotFound = errors.New("not found")
)

func isNotFound(err error) bool { return errors.Is(err, ErrNotFound) }

// ===== gRPC-СЕРВЕР =====

const maxBatch = 500

type Server struct {
	invpb.UnimplementedStockServiceServer
	q InventoryQueries
}

func NewServer(q InventoryQueries) *Server {
	return &Server{q: q}
}

func (s *Server) GetStock(ctx context.Context, req *invpb.GetStockRequest) (*invpb.GetStockResponse, error) {
	itemID := req.GetItemId()

	if itemID <= 0 {
		return nil, status.Error(codes.InvalidArgument, "item_id must be > 0")
	}
	location := req.GetLocationCode() // может быть пустой

	st, err := s.q.GetStock(ctx, itemID, location)
	if err != nil {
		if isNotFound(err) {
			return nil, status.Error(codes.NotFound, "item not found")
		}
		return nil, status.Errorf(codes.Internal, "get stock failed: %v", err)
	}

	pb := toPBStock(st, location)
	return &invpb.GetStockResponse{Stock: pb}, nil
}

func (s *Server) BatchGetStock(ctx context.Context, req *invpb.BatchGetStockRequest) (*invpb.BatchGetStockResponse, error) {
	ids := req.GetItemIds()
	if l := len(ids); l == 0 {
		return nil, status.Error(codes.InvalidArgument, "item_ids is empty")
	} else if l > maxBatch {
		return nil, status.Errorf(codes.InvalidArgument, "too many item_ids: %d > %d", l, maxBatch)
	}
	location := req.GetLocationCode()

	// Можно дедуплицировать, чтобы не грузить usecase лишними повторами.
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, status.Error(codes.InvalidArgument, "item_id must be > 0")
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			unique = append(unique, id)
		}
	}

	stocks, err := s.q.BatchGetStock(ctx, unique, location)
	if err != nil {
		if isNotFound(err) {
			// Конвенция: если хотя бы один из запрошенных отсутствует — NOT_FOUND.
			return nil, status.Error(codes.NotFound, "some item_id not found")
		}
		return nil, status.Errorf(codes.Internal, "batch get stock failed: %v", err)
	}

	// Сложим в map для быстрого доступа.
	byID := make(map[int64]StockDTO, len(stocks))
	for _, st := range stocks {
		byID[st.ItemID] = st
	}

	// Сформируем ответ в ТОЧНОМ порядке входных ids (с учётом повторов).
	out := make([]*invpb.Stock, 0, len(ids))
	for _, id := range ids {
		st, ok := byID[id]
		if !ok {
			// Если usecase вернул не для всех — считаем это логической ошибкой уровня app.
			return nil, status.Errorf(codes.Internal, "missing stock for item_id=%d in app response", id)
		}
		out = append(out, toPBStock(st, location))
	}

	return &invpb.BatchGetStockResponse{Stocks: out}, nil
}

// ===== МАППИНГ В PROTO =====

func toPBStock(s StockDTO, filterLocation string) *invpb.Stock {
	pb := &invpb.Stock{
		ItemId:    s.ItemID,
		Available: s.Available,
		UpdatedAt: toProtoTs(s.UpdatedAt),
	}

	// Если попросили конкретную локацию — отфильтруем и пересчитаем available.
	if filterLocation != "" {
		var only *StockPerLocationDTO
		for _, loc := range s.Locations {
			if loc.LocationCode == filterLocation {
				only = &loc
				break
			}
		}
		if only != nil {
			pb.Locations = []*invpb.StockPerLocation{toPBLocation(*only)}
			pb.Available = only.Available
		} else {
			// Нет такой локации у товара — считаем available=0 и пустой список.
			pb.Locations = nil
			pb.Available = 0
		}
		return pb
	}

	// Иначе — отдадим все локации как есть.
	pb.Locations = make([]*invpb.StockPerLocation, 0, len(s.Locations))
	for _, loc := range s.Locations {
		pb.Locations = append(pb.Locations, toPBLocation(loc))
	}
	return pb
}

func toPBLocation(l StockPerLocationDTO) *invpb.StockPerLocation {
	return &invpb.StockPerLocation{
		LocationCode: l.LocationCode,
		Available:    l.Available,
		UpdatedAt:    toProtoTs(l.UpdatedAt),
	}
}

func toProtoTs(t time.Time) *timestamppb.Timestamp {
	// invpb.Timestamp — это алиас google.protobuf.Timestamp из сгенерённого пакета.
	// Обычно генератор кладёт её в well-known types; в реальном коде используем timestamppb.New(t).
	if t.IsZero() {
		return nil
	}
	// Здесь просто покажем безопасный конверт; в реальном проекте импортируй timestamppb.
	sec := t.Unix()
	nsec := int32(t.Sub(time.Unix(sec, 0)))
	return &timestamppb.Timestamp{Seconds: sec, Nanos: int32(nsec)}
}

// (опционально) удобный враппер для внутренних ошибок
func internalf(format string, a ...any) error {
	return status.Errorf(codes.Internal, fmt.Sprintf(format, a...))
}
