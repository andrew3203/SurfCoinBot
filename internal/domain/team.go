package domain

type Team struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}
