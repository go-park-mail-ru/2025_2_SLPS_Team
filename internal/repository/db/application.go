package db

import (
	"context"
	"database/sql"
	"log"
	"project/domain"
)

type DBApplicationStore struct {
	db *sql.DB
}

func NewDBApplicationStore(db *sql.DB) domain.ApplicationStore {
	return &DBApplicationStore{db: db}
}

// Получение обращений с пагинацией для админа
func (r *DBApplicationStore) GetApplications(ctx context.Context, limit, offset int32) ([]domain.Application, error) {
	query := `
        SELECT id, author_id, text, category, status, created_at, updated_at, email_req, email_feedback
        FROM applications
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var app domain.Application
		err = rows.Scan(&app.ID, &app.AuthorID, &app.Text, &app.Category, &app.Status, &app.CreatedAt, &app.UpdatedAt, &app.EmailReg, &app.EmailFeedBack)
		if err != nil {
			return nil, err
		}

		apps = append(apps, app)
	}

	return apps, nil
}

func (r *DBApplicationStore) GetApplicationsByUser(ctx context.Context, limit, offset int32) ([]domain.Application, error) {
	query := `
        SELECT id, author_id, text, category, status, created_at, updated_at, email_req, email_feedback
        FROM applications
        WHERE author_id=$1 OR temp_session_id =$2
        ORDER BY created_at DESC
        LIMIT $3 OFFSET $4`
	TempSessionInfo, _ := ctx.Value(domain.TempSessionCtxKey).(*domain.TempSessionInfo)
	if TempSessionInfo == nil {
		TempSessionInfo = &domain.TempSessionInfo{}
	}
	log.Println(TempSessionInfo)
	rows, err := r.db.QueryContext(ctx, query, TempSessionInfo.UserID, TempSessionInfo.TempSessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []domain.Application
	for rows.Next() {
		var app domain.Application
		err = rows.Scan(&app.ID, &app.AuthorID, &app.Text, &app.Category, &app.Status, &app.CreatedAt, &app.UpdatedAt, &app.EmailReg, &app.EmailFeedBack)
		if err != nil {
			return nil, err
		}

		apps = append(apps, app)
	}

	return apps, nil
}

func (r *DBApplicationStore) UpdateApplicationText(ctx context.Context, id int32, newText string) error {
	query := `
        UPDATE applications 
        SET text = $1 
        WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newText, id)
	return err
}

func (r *DBApplicationStore) UpdateApplicationStatus(ctx context.Context, id int32, newStatus string) error {
	query := `
        UPDATE applications 
        SET status = $1
        WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, newStatus, id)
	return err
}

func (r *DBApplicationStore) CreateApplication(ctx context.Context, app domain.Application) (int32, error) {
	query := `
        INSERT INTO applications (author_id, temp_session_id, text, category, email_req, email_feedback)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id`

	TempSessionInfo, _ := ctx.Value(domain.TempSessionCtxKey).(*domain.TempSessionInfo)
	if TempSessionInfo == nil {
		TempSessionInfo = &domain.TempSessionInfo{}
	}
	var id int32
	err := r.db.QueryRowContext(ctx, query,
		TempSessionInfo.UserID,
		TempSessionInfo.TempSessionID,
		app.Text,
		app.Category,
		app.EmailReg,
		app.EmailFeedBack,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
func (r *DBApplicationStore) MergeTempSession(ctx context.Context) error {
	TempSessionInfo, _ := ctx.Value(domain.TempSessionCtxKey).(*domain.TempSessionInfo)
	if TempSessionInfo == nil {
		TempSessionInfo = &domain.TempSessionInfo{}
	}
	query := `
        UPDATE applications
        SET author_id = $1,
            temp_session_id = NULL
        WHERE temp_session_id = $2
    `
	_, err := r.db.Exec(query, TempSessionInfo.UserID, TempSessionInfo.TempSessionID)
	return err
}
