package repository

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"surf_bot/internal/domain"
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

	_, err = tx.Exec("INSERT INTO users (id, name, role) VALUES ($1, $2, $3)", user.ID, user.Name, user.Role)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert into users: %w", err)
	}

	if user.role != domain.RoleCoach {
		_, err = tx.Exec("INSERT INTO user_score (user_id, score) VALUES ($1, 0)", user.ID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert into user_score: %w", err)
	}
	}

	return tx.Commit()
}

type ScoreEntry struct {
	UserID int64  `db:"user_id"`
	Name   string `db:"name"`
	Score  int    `db:"score"`
}

// GetRanking returns athletes ordered by score DESC
func (r *UserRepository) GetRanking() ([]ScoreEntry, error) {
	query := `
		SELECT u.id as user_id, u.name, s.score
		FROM users u
		JOIN user_score s ON u.id = s.user_id
		WHERE u.role = 'athlete'
		ORDER BY s.score DESC, u.name ASC
	`

	var ranking []ScoreEntry
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
	ID     int    `db:"id"`
	UserID int64  `db:"from_id"`
	Name   string `db:"name"`
	Amount int    `db:"amount"`
	Reason string `db:"reason"`
}

// GetPendingRequests returns all pending point requests
func (r *UserRepository) GetPendingRequests() ([]PendingRequest, error) {
	query := `
		SELECT p.id, p.from_id, u.name, p.amount, p.reason
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
		tx.Rollback()
		return fmt.Errorf("запрос не найден или уже обработан: %w", err)
	}

	// Начислить баллы
	_, err = tx.Exec("UPDATE user_score SET score = score + $1 WHERE user_id = $2", req.Amount, req.FromID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("не удалось начислить баллы: %w", err)
	}

	// Пометить как подтвержденный
	_, err = tx.Exec("UPDATE point SET pending = false WHERE id = $1", id)
	if err != nil {
		tx.Rollback()
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
		tx.Rollback()
		return fmt.Errorf("спортсмен с именем %s не найден: %w", toUsername, err)
	}

	// начислить баллы
	_, err = tx.Exec(`
		UPDATE user_score SET score = score + $1 WHERE user_id = $2
	`, amount, user.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("не удалось начислить баллы: %w", err)
	}

	// сохранить в point
	_, err = tx.Exec(`
		INSERT INTO point (from_id, amount, reason, pending)
		VALUES ($1, $2, $3, false)
	`, user.ID, amount, reason)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("не удалось сохранить в историю: %w", err)
	}

	return tx.Commit()
}

type AthleteShort struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
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

type ScoreEntry struct {
	Amount int    `db:"amount"`
	Reason string `db:"reason"`
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
