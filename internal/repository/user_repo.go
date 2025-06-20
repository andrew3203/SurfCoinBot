package repository

import (
	"database/sql"
	"fmt"

	"surf_bot/internal/domain"
	"surf_bot/internal/util"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	DB *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// GetUserByID returns a user if exists
func (r *UserRepository) GetUserByID(id int64) (*domain.User, error) {
	var user domain.User
	err := r.DB.Get(&user, "SELECT id, name, role FROM users WHERE id = $1", id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// RegisterUser inserts user and score if not exists
func (r *UserRepository) RegisterUser(user *domain.User) error {
	existing, err := r.GetUserByID(user.ID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // already exists
	}

	tx := r.DB.MustBegin()

	_, err = tx.Exec("INSERT INTO users (id, name, username, role) VALUES ($1, $2, $3, $4)", user.ID, user.Name, user.Username, user.Role)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("failed to insert into users: %w", err)
	}

	if user.Role != domain.RoleCoach {
		_, err = tx.Exec("INSERT INTO user_score (user_id, score) VALUES ($1, 0)", user.ID)
		if err != nil {
			util.SafeRollback(tx)
			return fmt.Errorf("failed to insert into user_score: %w", err)
		}
	}

	return tx.Commit()
}

// GetRanking returns athletes ordered by score DESC
func (r *UserRepository) GetRanking() ([]domain.ScoreEntry, error) {
	query := `
		SELECT u.id as user_id, u.name, s.score
		FROM users u
		JOIN user_score s ON u.id = s.user_id
		WHERE u.role = 'athlete'
		ORDER BY s.score DESC, u.name ASC
	`

	var ranking []domain.ScoreEntry
	err := r.DB.Select(&ranking, query)
	if err != nil {
		return nil, err
	}

	return ranking, nil
}

// CreatePendingRequest stores a pending point request from athlete
func (r *UserRepository) CreatePendingRequest(fromID int64, amount int, reason string) error {
	_, err := r.DB.Exec(`
		INSERT INTO point (from_id, amount, reason, pending)
		VALUES ($1, $2, $3, true)
	`, fromID, amount, reason)

	if err != nil {
		return fmt.Errorf("failed to insert point request: %w", err)
	}
	return nil
}

type PendingRequest struct {
	ID       int    `db:"id"`
	UserID   int64  `db:"from_id"`
	Name     string `db:"name"`
	Username string `db:"username"`
	Amount   int    `db:"amount"`
	Reason   string `db:"reason"`
}

// GetPendingRequests returns all pending point requests
func (r *UserRepository) GetPendingRequests() ([]PendingRequest, error) {
	query := `
		SELECT p.id, p.from_id, u.name, u.username, p.amount, p.reason
		FROM point p
		JOIN users u ON p.from_id = u.id
		WHERE p.pending = true
		ORDER BY p.id ASC
	`

	var requests []PendingRequest
	err := r.DB.Select(&requests, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pendings data: %w", err)
	}

	return requests, nil
}

func (r *UserRepository) ApproveRequest(id int) error {
	tx := r.DB.MustBegin()

	var req struct {
		FromID int64 `db:"from_id"`
		Amount int   `db:"amount"`
	}

	// Найти запрос
	err := tx.Get(&req, "SELECT from_id, amount FROM point WHERE id = $1 AND pending = true", id)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("запрос не найден или уже обработан: %w", err)
	}

	// Начислить баллы
	_, err = tx.Exec("UPDATE user_score SET score = score + $1 WHERE user_id = $2", req.Amount, req.FromID)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("не удалось начислить баллы: %w", err)
	}

	// Пометить как подтвержденный
	_, err = tx.Exec("UPDATE point SET pending = false WHERE id = $1", id)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("не удалось обновить статус запроса: %w", err)
	}

	return tx.Commit()
}

func (r *UserRepository) GivePoints(toUsername string, amount int, reason string) error {
	tx := r.DB.MustBegin()

	// найти пользователя по username (без @)
	var user struct {
		ID int64 `db:"id"`
	}
	err := tx.Get(&user, `
		SELECT id FROM users
		WHERE LOWER(username) = LOWER($1) AND role = 'athlete'
	`, toUsername)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("спортсмен с именем %s не найден: %w", toUsername, err)
	}

	// начислить баллы
	_, err = tx.Exec(`
		UPDATE user_score SET score = score + $1 WHERE user_id = $2
	`, amount, user.ID)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("не удалось начислить баллы: %w", err)
	}

	// сохранить в point
	_, err = tx.Exec(`
		INSERT INTO point (from_id, amount, reason, pending)
		VALUES ($1, $2, $3, false)
	`, user.ID, amount, reason)
	if err != nil {
		util.SafeRollback(tx)
		return fmt.Errorf("не удалось сохранить в историю: %w", err)
	}

	return tx.Commit()
}

type AthleteShort struct {
	ID       int64  `db:"id"`
	Name     string `db:"name"`
	Username string `db:"username"`
}

func (r *UserRepository) ListAthletes() ([]AthleteShort, error) {
	var athletes []AthleteShort
	err := r.DB.Select(&athletes, `
		SELECT id, name, username FROM users WHERE role = 'athlete' ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка спортсменов: %w", err)
	}
	return athletes, nil
}

func (r *UserRepository) RejectRequest(id int) (int64, error) {
	var fromID int64
	err := r.DB.Get(&fromID, `
		SELECT from_id FROM point WHERE id = $1 AND pending = true
	`, id)
	if err != nil {
		return 0, fmt.Errorf("запрос не найден или уже обработан: %w", err)
	}

	_, err = r.DB.Exec(`
		DELETE FROM point WHERE id = $1
	`, id)
	if err != nil {
		return 0, fmt.Errorf("ошибка при удалении запроса: %w", err)
	}

	return fromID, nil
}

func (r *UserRepository) GetUserByUsername(username string) (*domain.User, error) {
	var u domain.User
	err := r.DB.Get(&u, `
		SELECT id, name, username, role
		FROM users
		WHERE LOWER(username) = LOWER($1)
	`, username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetUserScore(userID int64) (int, error) {
	var score int
	err := r.DB.Get(&score, `
		SELECT score FROM user_score WHERE user_id = $1
	`, userID)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить счёт: %w", err)
	}
	return score, nil
}

// GetUserHistory returns the list of confirmed point records for a given user.
func (r *UserRepository) GetUserHistory(userID int64) ([]domain.PointRecord, error) {
	var history []domain.PointRecord

	query := `
		SELECT amount, reason
		FROM point
		WHERE from_id = $1 AND pending = false
		ORDER BY id DESC
	`

	err := r.DB.Select(&history, query, userID)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить историю начислений: %w", err)
	}

	return history, nil
}

func (r *UserRepository) GetTeamByName(name string) (*domain.Team, error) {
	var team domain.Team
	err := r.DB.Get(&team, `SELECT id, name FROM team WHERE name = $1`, name)
	if err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *UserRepository) GetTeamByID(id int) (*domain.Team, error) {
	var team domain.Team
	err := r.DB.Get(&team, "SELECT id, name FROM team WHERE id = $1", id)
	if err != nil {
		return nil, fmt.Errorf("команда с ID %d не найдена: %w", id, err)
	}
	return &team, nil
}

func (r *UserRepository) AssignUserToTeam(userID int64, teamID int) error {
	_, err := r.DB.Exec("UPDATE users SET team_id = $1 WHERE id = $2", teamID, userID)
	if err != nil {
		return fmt.Errorf("не удалось назначить команду пользователю: %w", err)
	}
	return nil
}

func (r *UserRepository) CreateTeam(name string) error {
	_, err := r.DB.Exec(`INSERT INTO team (name) VALUES ($1)`, name)
	return err
}

func (r *UserRepository) DeleteTeam(teamID int) error {
	_, err := r.DB.Exec(`DELETE FROM team WHERE id = $1`, teamID)
	return err
}

func (r *UserRepository) ListTeams() ([]domain.Team, error) {
	var teams []domain.Team
	err := r.DB.Select(&teams, `
		SELECT id, name FROM team ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении команд: %w", err)
	}
	return teams, nil
}

func (r *UserRepository) ListAthletesByTeam(teamID *int) ([]domain.AthleteShort, error) {
	query := "SELECT id, name, username FROM users WHERE role = 'athlete'"
	var args []interface{}
	if teamID != nil {
		query += " AND team_id = $1"
		args = append(args, *teamID)
	}
	query += " ORDER BY name ASC"

	var athletes []domain.AthleteShort
	err := r.DB.Select(&athletes, query, args...)
	return athletes, err
}

func (r *UserRepository) GetPendingRequestsByTeam(teamID *int) ([]PendingRequest, error) {
	query := `
		SELECT p.id, p.from_id, u.name, u.username, p.amount, p.reason
		FROM point p
		JOIN users u ON p.from_id = u.id
		WHERE p.pending = true`
	var args []interface{}
	if teamID != nil {
		query += " AND u.team_id = $1"
		args = append(args, *teamID)
	}
	query += " ORDER BY p.id ASC"

	var requests []PendingRequest
	err := r.DB.Select(&requests, query, args...)
	return requests, err
}

func (r *UserRepository) GetRankingByTeam(teamID int) ([]domain.ScoreEntry, error) {
	query := `
		SELECT u.id as user_id, u.name, u.username, s.score
		FROM users u
		JOIN user_score s ON u.id = s.user_id
		WHERE u.role = 'athlete' AND u.team_id = $1
		ORDER BY s.score DESC, u.name ASC
	`
	var ranking []domain.ScoreEntry
	err := r.DB.Select(&ranking, query, teamID)
	if err != nil {
		return nil, err
	}
	return ranking, nil
}

func (r *UserRepository) GetUserTeamID(userID int64) (int, error) {
	var teamID int
	err := r.DB.Get(&teamID, `SELECT team_id FROM users WHERE id = $1`, userID)
	if err != nil {
		return 0, fmt.Errorf("не удалось получить team_id пользователя: %w", err)
	}
	return teamID, nil
}


func (r *UserRepository) GetUserTeamName(userID int64) (string, error) {
	var name string
	err := r.DB.Get(&name, `SELECT name FROM users WHERE id = $1`, userID)
	if err != nil {
		return "", fmt.Errorf("не удалось получить команду пользователя: %w", err)
	}
	return name, nil
}