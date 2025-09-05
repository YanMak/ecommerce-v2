package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"example.com/sqlchello/internal/adapters/outbound/postgres"
	repoports "example.com/sqlchello/internal/core/ports/repo"
	"example.com/sqlchello/internal/core/usecase/paging"

	"example.com/sqlchello/internal/core/usecase"
	ptr "example.com/sqlchello/internal/x"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/shop?sslmode=disable")

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("pgxpool:", err)
	}
	defer pool.Close()

	repo := postgres.New(pool)          // Outbound Adapter
	svc := usecase.NewItemService(repo) // Use Case

	// // 1) Создаём
	// item, err := svc.Create(ctx, domain.Item{
	// 	Slug:        fmt.Sprintf("demo-%d", time.Now().UnixNano()),
	// 	Name:        "Demo",
	// 	Description: "from hex + sqlc",
	// 	PriceCents:  2599,
	// 	Tags:        []string{"sqlc", "hex"},
	// })
	// if err != nil {
	// 	log.Fatal("create:", err)
	// }
	// log.Println("created:", item.ID, item.Slug)

	// // 2) Читаем
	// got, err := svc.Get(ctx, item.ID)
	// if err != nil {
	// 	log.Fatal("get:", err)
	// }
	// log.Println("fetched:", got.Name, got.PriceCents)

	// //name := "Demo"                                    // попробуй nil вместо &name
	// //min := int64(2000) // попробуй nil вместо &min
	// max := int64(2000) // попробуй nil вместо &min
	// //items, err := svc.Search(ctx, &name, &min, 10, 0) // или (nil, nil, 10, 0)
	// items, err := svc.Search(ctx, nil, nil, &max, 10, 0) // или (nil, nil, 10, 0)
	// if err != nil {
	// 	log.Fatal("search:", err)
	// }
	// log.Println("search result:", len(items))
	// for _, it := range items {
	// 	log.Println(" -", it.ID, it.Name, it.PriceCents)
	// }

	// ///////////
	// // 1) Создаём
	// items, err := svc.CreateAndSearch(ctx, domain.Item{
	// 	Slug:        fmt.Sprintf("demo-%d", time.Now().UnixNano()),
	// 	Name:        "Demo",
	// 	Description: "from hex + sqlc",
	// 	PriceCents:  2599,
	// 	Tags:        []string{"sqlc", "hex"},
	// },
	// 	nil, nil, nil, 10, 0,
	// )
	// if err != nil {
	// 	log.Fatal("create:", err)
	// }
	// log.Printf("searched: %+v", items)

	// slug := fmt.Sprintf("demo-%d", time.Now().UnixNano())
	// items, err := svc.CreateAndSearchWithRetry(ctx, domain.Item{
	// 	//Slug:        fmt.Sprintf("demo-%d", time.Now().UnixNano()),
	// 	Slug:        slug,
	// 	Name:        "Demo",
	// 	Description: "from hex + sqlc",
	// 	PriceCents:  2599,
	// 	Tags:        []string{"sqlc", "hex"},
	// },
	// 	nil, nil, nil, 10, 0,
	// )
	// if err != nil {
	// 	log.Fatal("create:", err)
	// }
	// log.Printf("searched: %+v", items)

	// // Create + Update
	// ctx = context.Background()
	// slug = fmt.Sprintf("demo-%d", time.Now().UnixNano())
	// created, err := svc.Create(ctx, domain.Item{
	// 	//Slug:        fmt.Sprintf("demo-%d", time.Now().UnixNano()),
	// 	Slug:        slug,
	// 	Name:        "Demo",
	// 	Description: "from hex + sqlc",
	// 	PriceCents:  2599,
	// 	Tags:        []string{"sqlc", "hex"},
	// })

	// var priceCents int64 = 2699
	// updated, err := svc.Patch(ctx, created.ID, ports.ItemPatch{
	// 	//Slug:        fmt.Sprintf("demo-%d", time.Now().UnixNano()),
	// 	Name:          nil,
	// 	Description:   ptr.To("from hex + sqlc updated"),
	// 	PriceCents:    ptr.To(priceCents),
	// 	Tags:          ptr.To([]string{"sqlc", "hex", "added tag"}),
	// 	PrevUpdatedAt: &created.UpdatedAt,
	// })
	// fmt.Println("updated ", updated)

	// upserted, err := svc.UpsertBySlug(ctx, domain.Item{
	// 	Slug:        updated.Slug,
	// 	Name:        "name from upsert",
	// 	Description: "from hex + sqlc upserted",
	// 	PriceCents:  2799,
	// 	Tags:        []string{"sqlc", "hex", "added tag", "upserted tag"},
	// })
	// fmt.Println("upserted ", upserted)

	var priceFrom, priceTo int64
	priceFrom = 1
	priceTo = 50000
	page := int32(1)
	perPage := int32(5)
	offsetResult, err := svc.SearchOffset(ctx,
		repoports.SearchFilter{
			//Name:     ptr.To("emo"),
			MinPrice: ptr.To(priceFrom),
			MaxPrice: ptr.To(priceTo),
		}, paging.OffsetPage{Page: page, PerPage: perPage})
	fmt.Printf("offsetResult page %d: %+v\n", page, offsetResult)
	for currPage := page + 1; currPage*perPage < int32(offsetResult.Total); currPage++ {

		offsetResult, err := svc.SearchOffset(ctx,
			repoports.SearchFilter{
				//Name:     ptr.To("emo"),
				MinPrice: ptr.To(priceFrom),
				MaxPrice: ptr.To(priceTo),
			}, paging.OffsetPage{Page: currPage, PerPage: perPage})
		fmt.Printf("offsetResult page %d: %+v, err=%s\n\n\n", currPage, offsetResult, err)
	}

	// FRONT DIRECTION
	currPageFront := 1
	keysetResult, err := svc.SearchKeysetNext(ctx,
		repoports.SearchFilter{
			//Name:     ptr.To("emo"),
			MinPrice: ptr.To(priceFrom),
			MaxPrice: ptr.To(priceTo),
		}, 5, nil)
	fmt.Printf("front direction keysetResult, page %d: %+v\n cursor=%+v err=%s \n\n\n", currPageFront, keysetResult, keysetResult.Cursor, err)

	var next *paging.Cursor = keysetResult.Cursor
	for keysetResult.HasNext {
		// FRONT DIRECTION
		currPageFront++
		keysetResult, err := svc.SearchKeysetNext(ctx,
			repoports.SearchFilter{
				//Name:     ptr.To("emo"),
				MinPrice: ptr.To(priceFrom),
				MaxPrice: ptr.To(priceTo),
			}, 5, next)
		fmt.Printf("front direction  keysetResult, page %d: %+v\n cursor=%+v err=%s \n\n\n", currPageFront, keysetResult, keysetResult.Cursor, err)
		if !keysetResult.HasNext {
			break
		}
		next = keysetResult.Cursor

	}

	// BACK DIRECTION
	currPageBack := 1
	keysetResult, err = svc.SearchKeysetPrev(ctx,
		repoports.SearchFilter{
			//Name:     ptr.To("emo"),
			MinPrice: ptr.To(priceFrom),
			MaxPrice: ptr.To(priceTo),
		}, 5, next)
	fmt.Printf("back direction  keysetResult, page %d: %+v\n cursor=%+v err=%s \n\n\n", currPageBack, keysetResult, keysetResult.Cursor, err)

	prev := keysetResult.Cursor
	for keysetResult.HasNext {
		currPageBack++
		keysetResult, err := svc.SearchKeysetPrev(ctx,
			repoports.SearchFilter{
				//Name:     ptr.To("emo"),
				MinPrice: ptr.To(priceFrom),
				MaxPrice: ptr.To(priceTo),
			}, 5, prev)
		fmt.Printf("back direction keysetResult, page %d: %+v\n cursor=%+v err=%s \n\n\n", currPageBack, keysetResult, keysetResult.Cursor, err)
		if !keysetResult.HasNext {
			break
		}
		prev = keysetResult.Cursor
	}

}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
