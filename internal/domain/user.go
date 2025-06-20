package domain

type Role string

const (
	RoleAthlete Role = "athlete"
	RoleCoach   Role = "coach"
)

type User struct {
	ID   int64
	Name string
	Username string
	Role Role
}
