package inventory

// # переиспользуемые SDK-клиенты к сервисам
// │  └─ inventory/                 # Catalog, Order и др. берут один и тот же клиент
// │     └─ client.go               # (наш outbound-клиент сюда перенесём)
// ├─ pkg/                          # инфра-утилиты (без домена)
