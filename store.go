package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type User struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

type Student struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Lesson struct {
	ID              string `json:"id"`
	StudentID       string `json:"studentId"`
	StudentName     string `json:"studentName"`
	Date            string `json:"date"`
	StartTime       string `json:"startTime"`
	DurationMinutes int    `json:"durationMinutes"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

func NewStore(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	statements := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE COLLATE NOCASE,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);`,
		`CREATE TABLE IF NOT EXISTS students (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_students_user ON students(user_id, name);`,
		`CREATE TABLE IF NOT EXISTS lessons (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			student_id TEXT NOT NULL REFERENCES students(id) ON DELETE RESTRICT,
			lesson_date TEXT NOT NULL,
			start_time TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_lessons_user_date ON lessons(user_id, lesson_date);`,
		`CREATE INDEX IF NOT EXISTS idx_lessons_student ON lessons(student_id, lesson_date);`,
	}

	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("数据库初始化失败: %w", err)
		}
	}
	if err := s.dropLessonStatusColumn(ctx); err != nil {
		return fmt.Errorf("数据库初始化失败: %w", err)
	}

	return nil
}

func (s *Store) dropLessonStatusColumn(ctx context.Context) error {
	var count int
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM pragma_table_info('lessons') WHERE name = 'status'`,
	).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `ALTER TABLE lessons DROP COLUMN status`)
	return err
}

func (s *Store) userCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (s *Store) createUser(ctx context.Context, user User, passwordHash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO users (id, name, password_hash, created_at)
		 VALUES (?, ?, ?, ?)`,
		user.ID, user.Name, passwordHash, user.CreatedAt,
	)
	return err
}

func (s *Store) findUserByName(ctx context.Context, name string) (User, string, error) {
	var user User
	var passwordHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, password_hash, created_at
		 FROM users WHERE name = ? COLLATE NOCASE`,
		name,
	).Scan(&user.ID, &user.Name, &passwordHash, &user.CreatedAt)
	return user, passwordHash, err
}

func (s *Store) findUserByID(ctx context.Context, id string) (User, error) {
	var user User
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, created_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Name, &user.CreatedAt)
	return user, err
}

func (s *Store) listStudents(ctx context.Context, userID, search string) ([]Student, error) {
	query := `SELECT id, name
		FROM students WHERE user_id = ? AND deleted_at IS NULL`
	args := []any{userID}
	if search != "" {
		query += ` AND name LIKE ?`
		like := "%" + search + "%"
		args = append(args, like)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := make([]Student, 0)
	for rows.Next() {
		var student Student
		if err := rows.Scan(&student.ID, &student.Name); err != nil {
			return nil, err
		}
		students = append(students, student)
	}
	return students, rows.Err()
}

func (s *Store) createStudent(ctx context.Context, userID string, student Student) (Student, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	var deletedStudentID string
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM students
		 WHERE user_id = ? AND name = ? COLLATE NOCASE AND deleted_at IS NOT NULL
		 ORDER BY updated_at DESC, created_at DESC
		 LIMIT 1`,
		userID, student.Name,
	).Scan(&deletedStudentID)
	if err == nil {
		_, err = s.db.ExecContext(ctx,
			`UPDATE students SET name = ?, deleted_at = NULL, updated_at = ?
			 WHERE id = ? AND user_id = ?`,
			student.Name, now, deletedStudentID, userID,
		)
		if err != nil {
			return Student{}, err
		}
		student.ID = deletedStudentID
		return student, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Student{}, err
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO students (id, user_id, name, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?)`,
		student.ID, userID, student.Name, now, now,
	)
	return student, err
}

func (s *Store) updateStudent(ctx context.Context, userID, studentID string, student Student) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE students SET name = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		student.Name, time.Now().UTC().Format(time.RFC3339), studentID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (s *Store) deleteStudent(ctx context.Context, userID, studentID string) (bool, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx,
		`UPDATE students SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		now, now, studentID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}
	return true, nil
}

func (s *Store) studentBelongsToUser(ctx context.Context, userID, studentID string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx,
		`SELECT 1 FROM students WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		studentID, userID,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) createLesson(ctx context.Context, userID string, lesson Lesson) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO lessons
		 (id, user_id, student_id, lesson_date, start_time, duration_minutes, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		lesson.ID, userID, lesson.StudentID, lesson.Date, lesson.StartTime, lesson.DurationMinutes,
		now, now,
	)
	return err
}

func (s *Store) updateLesson(ctx context.Context, userID, lessonID string, lesson Lesson) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE lessons SET student_id = ?, lesson_date = ?, start_time = ?, duration_minutes = ?, updated_at = ?
		 WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		lesson.StudentID, lesson.Date, lesson.StartTime, lesson.DurationMinutes,
		time.Now().UTC().Format(time.RFC3339), lessonID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (s *Store) deleteLesson(ctx context.Context, userID, lessonID string) (bool, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE lessons SET deleted_at = ?, updated_at = ? WHERE id = ? AND user_id = ? AND deleted_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), lessonID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (s *Store) listLessons(ctx context.Context, userID, month, studentID string) ([]Lesson, error) {
	query := `SELECT l.id, l.student_id, s.name, l.lesson_date, l.start_time,
		l.duration_minutes, l.created_at, l.updated_at
		FROM lessons l JOIN students s ON s.id = l.student_id
		WHERE l.user_id = ? AND l.deleted_at IS NULL AND s.deleted_at IS NULL`
	args := []any{userID}
	if month != "" {
		query += ` AND substr(l.lesson_date, 1, 7) = ?`
		args = append(args, month)
	}
	if studentID != "" {
		query += ` AND l.student_id = ?`
		args = append(args, studentID)
	}
	query += ` ORDER BY l.lesson_date DESC, l.start_time DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lessons := make([]Lesson, 0)
	for rows.Next() {
		var lesson Lesson
		if err := rows.Scan(
			&lesson.ID, &lesson.StudentID, &lesson.StudentName,
			&lesson.Date, &lesson.StartTime, &lesson.DurationMinutes,
			&lesson.CreatedAt, &lesson.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lessons = append(lessons, lesson)
	}
	return lessons, rows.Err()
}

func (s *Store) getLesson(ctx context.Context, userID, lessonID string) (Lesson, error) {
	var lesson Lesson
	err := s.db.QueryRowContext(ctx,
		`SELECT l.id, l.student_id, s.name, l.lesson_date, l.start_time,
			l.duration_minutes, l.created_at, l.updated_at
		 FROM lessons l JOIN students s ON s.id = l.student_id
		 WHERE l.id = ? AND l.user_id = ? AND l.deleted_at IS NULL AND s.deleted_at IS NULL`,
		lessonID, userID,
	).Scan(
		&lesson.ID, &lesson.StudentID, &lesson.StudentName,
		&lesson.Date, &lesson.StartTime, &lesson.DurationMinutes,
		&lesson.CreatedAt, &lesson.UpdatedAt,
	)
	return lesson, err
}
