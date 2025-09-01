package grpcstock

type StockDTO struct {
	ItemID    int64
	Available int64
	Locations []StockPerLocationDTO
	UpdatedAt int64 // unix seconds
}

type StockPerLocationDTO struct {
	LocationCode string
	Available    int64
	UpdatedAt    int64 // unix seconds
}
