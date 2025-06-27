package dbhelper

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/ray-remotestate/restro/database"
	"github.com/ray-remotestate/restro/models"
)

type SQLExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func CreateUser(tx *sql.Tx, name, email, hashedPassword string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(`INSERT INTO users (name, email, password, created_by) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, email, hashedPassword, uuid.Nil).Scan(&id)
	return id, err
}

func IsUserExists(email string) (bool, error) {
	var count int
	err := database.Restro.QueryRow(`SELECT COUNT(*) FROM users WHERE LOWER(email) = LOWER($1)`, email).Scan(&count)
	return count > 0, err
}

func AssignRole(tx *sql.Tx, userID uuid.UUID, role models.Role) error {
	_, err := tx.Exec(`INSERT INTO user_roles (user_id, role) VALUES ($1, $2)`, userID, role)
	return err
}

func GetUserByEmail(email string) (uuid.UUID, error) {
	var userID uuid.UUID

	err := database.Restro.QueryRow(`
		SELECT id FROM users
		WHERE email = $1 AND archived_at IS NULL`, email).
		Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func GetUserByPassword(email, password string) (uuid.UUID, string, error) {
	var id uuid.UUID
	var hashedPassword string
	var name string

	err := database.Restro.QueryRow(`
		SELECT id, name, password FROM users 
		WHERE LOWER(email) = LOWER($1) AND archived_at IS NULL`, email).
		Scan(&id, &name, &hashedPassword)
	if err != nil {
		return uuid.Nil, "", err
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
		return uuid.Nil, "", fmt.Errorf("incorrect password")
	}

	return id, name, nil
}

func GetUserRoleByUserID(userID uuid.UUID) (*sql.Rows, error) {
	rows, err := database.Restro.Query(`
		SELECT role FROM user_roles
		WHERE user_id = $1 AND archived_at IS NULL`, userID)
	if err != nil {
		return &sql.Rows{}, err
	}

	return rows, nil
}

func IsSubAdmin(id uuid.UUID) (bool, error) {
	var roleExists bool
	err := database.Restro.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM user_roles
			WHERE user_id = $1 AND role = 'subadmin' AND archived_at IS NULL
		)`, id).Scan(&roleExists)
	if err != nil{
		return false, err
	}

	return roleExists, nil
}

func MakeSubAdmin(id uuid.UUID) error {
	_, err := database.Restro.Exec(`
		INSERT INTO user_roles (user_id, role)
		VALUES ($1, 'subadmin')`, id)
	return err
}

func ListAllSubadmins() (*sql.Rows, error){
	rows, err := database.Restro.Query(`
		SELECT u.id, u.name, u.email
		FROM users u
		JOIN user_roles ur ON u.id = ur.user_id
		WHERE ur.role = 'subadmin' AND u.archived_at IS NULL AND ur.archived_at IS NULL`)

	if err != nil {
		return &sql.Rows{}, err
	}
	return rows, nil
}