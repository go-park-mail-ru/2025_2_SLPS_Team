package db

import (
	"database/sql"
	"project/domain"
	"time"
)

type DBUserStore struct {
	db *sql.DB
}

func NewDBUserStore(db *sql.DB) domain.UserStore {
	return &DBUserStore{db: db}
}

func (store *DBUserStore) AddUser(firstname, lastname, email, gender, password string, dob time.Time) (string, error) {
	query := `INSERT INTO profile 
                        (first_name, last_name, email, gender, password, dob) 
                        VALUES ($1,$2,$3,$4,$5,$6)`
	_, err := store.db.Exec(query, firstname, lastname, email, gender, password, dob)
	if err != nil {
		return "", err
	}
	return email, nil
}

func (store *DBUserStore) GetUserByEmail(email string) (domain.User, bool) {
	query := `SELECT id, first_name,last_name,dob, email, password FROM profile WHERE email = $1`

	var user domain.User
	err := store.db.QueryRow(query, email).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Password)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.User{}, false
		}
		return domain.User{}, false
	}

	return user, true
}
