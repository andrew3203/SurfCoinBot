package domain

type PointRecord struct {
	Amount int    `db:"amount"`
	Reason string `db:"reason"`
}
