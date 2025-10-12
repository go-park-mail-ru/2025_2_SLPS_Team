package db

import (
	"database/sql"
	"errors"
	"log"
	"project/domain"
)

type DBProfileStore struct {
	db *sql.DB
}

func NewDBProfileStore(db *sql.DB) domain.ProfileStore {
	return &DBProfileStore{db: db}
}

func (store *DBProfileStore) UpdateProfile(profile domain.Profile, userID int) error {
	log.Println(userID)
	queryProfile := `UPDATE profiles SET first_name = $2, last_name = $3, gender = $4, dob = $5, about_myself = $6
WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile,
		userID,
		profile.FirstName,
		profile.LastName,
		profile.Gender,
		profile.Dob,
		profile.AboutMyself)
	return err
}

func (store *DBProfileStore) UpdateAvatar(avatarPath string, userID int) error {
	queryProfile := `UPDATE profiles SET  avatar_path = $2
WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile, userID, avatarPath)
	return err
}

func (store *DBProfileStore) UpdateHeader(headerPath string, userID int) error {
	queryProfile := `UPDATE profiles SET  header_path = $2
WHERE user_id = $1`
	_, err := store.db.Exec(queryProfile, userID, headerPath)
	return err
}

func (store *DBProfileStore) GetProfileByUserID(userID int) (domain.Profile, error) {
	query := `SELECT user_id, first_name, last_name, avatar_path, header_path, about_myself, gender, dob  FROM profiles WHERE user_id = $1`
	//добавить null проверку легче всего просто добавить указатели на возможные нул поля
	// можно возвращать указатель на объект так будет проще понять что его нет
	var profile domain.Profile
	err := store.db.QueryRow(query, userID).Scan(
		&profile.UserID,
		&profile.FirstName,
		&profile.LastName,
		&profile.AvatarPath,
		&profile.HeaderPath,
		&profile.AboutMyself,
		&profile.Gender,
		&profile.Dob)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Profile{}, domain.ErrNotFound
		}
		return domain.Profile{}, err
	}

	return profile, nil
}

func (store *DBProfileStore) GetAvatarByUserID(userID int) (*string, error) {
	query := `SELECT avatar_path FROM profiles WHERE user_id = $1`
	var avatar *string
	err := store.db.QueryRow(query, userID).Scan(
		&avatar)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return avatar, nil
}

func (store *DBProfileStore) GetHeaderByUserID(userID int) (*string, error) {
	query := `SELECT header_path FROM profiles WHERE user_id = $1`
	var header *string
	err := store.db.QueryRow(query, userID).Scan(
		&header)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return header, nil
}
