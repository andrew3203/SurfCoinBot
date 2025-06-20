package domain

type ScoreEntry struct {
	UserID   int64  `db:"user_id"`
	Name     string `db:"name"`
	Username string `db:"username"`
	Score    int    `db:"score"`
}
