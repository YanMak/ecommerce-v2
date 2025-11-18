package postgres

import (
	"time"
)

type TxExtra struct {
	Retries     int           // сколько попыток всего (включая первую), обычно 2–3
	BaseBackoff time.Duration // стартовая пауза перед 2-й попыткой (напр. 50ms)
	MaxBackoff  time.Duration // потолок бэкоффа (напр. 1s)
	StmtTimeout time.Duration // SET LOCAL statement_timeout
	LockTimeout time.Duration // SET LOCAL lock_timeout
	IdleInTx    time.Duration // SET LOCAL idle_in_transaction_session_timeout
}
