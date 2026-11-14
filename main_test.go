package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := hashPassword("drumtest123")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(hash, "drumtest123") {
		t.Fatal("correct password did not verify")
	}
	if verifyPassword(hash, "wrongpass") {
		t.Fatal("incorrect password verified")
	}
}

func TestLessonIsCompleted(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, time.November, 11, 12, 0, 0, 0, location)
	tests := []struct {
		date     string
		start    string
		duration int
		want     bool
	}{
		{"2026-11-10", "18:00", 60, true},
		{"2026-11-11", "10:59", 60, true},
		{"2026-11-11", "11:00", 60, true},
		{"2026-11-11", "11:01", 60, false},
		{"2026-11-11", "18:00", 60, false},
		{"2026-11-12", "10:00", 60, false},
	}
	for _, test := range tests {
		if got := lessonIsCompleted(test.date, test.start, test.duration, now); got != test.want {
			t.Errorf(
				"lessonIsCompleted(%q, %q, %d) = %v, want %v",
				test.date,
				test.start,
				test.duration,
				got,
				test.want,
			)
		}
	}
}

func TestFreshDatabaseSchema(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "student-manager.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	expected := map[string][]string{
		"users":    {"id", "name", "password_hash", "created_at"},
		"students": {"id", "user_id", "name", "created_at", "updated_at", "deleted_at"},
		"lessons": {
			"id", "user_id", "student_id", "lesson_date", "start_time",
			"duration_minutes", "created_at", "updated_at", "deleted_at",
		},
	}
	for table, wanted := range expected {
		got, err := tableColumns(store, table)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, ",") != strings.Join(wanted, ",") {
			t.Fatalf("%s columns = %v, want %v", table, got, wanted)
		}
	}

	var attachmentTableCount int
	if err := store.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'attachments'`,
	).Scan(&attachmentTableCount); err != nil {
		t.Fatal(err)
	}
	if attachmentTableCount != 0 {
		t.Fatal("unused attachments table still exists")
	}
}

func TestLegacyLessonStatusColumnIsRemoved(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "student-manager.db")
	legacy, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`CREATE TABLE lessons (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		student_id TEXT NOT NULL,
		lesson_date TEXT NOT NULL,
		start_time TEXT NOT NULL,
		duration_minutes INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'scheduled',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		deleted_at TEXT
	)`); err != nil {
		legacy.Close()
		t.Fatal(err)
	}
	if _, err := legacy.Exec(`INSERT INTO lessons
		(id, user_id, student_id, lesson_date, start_time, duration_minutes, status, created_at, updated_at)
		VALUES ('les_legacy', 'usr_legacy', 'stu_legacy', '2026-11-10', '18:00', 60, 'scheduled', '', '')`); err != nil {
		legacy.Close()
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	columns, err := tableColumns(store, "lessons")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range columns {
		if column == "status" {
			t.Fatal("legacy status column was not removed")
		}
	}

	var lessonID string
	if err := store.db.QueryRow(`SELECT id FROM lessons`).Scan(&lessonID); err != nil {
		t.Fatal(err)
	}
	if lessonID != "les_legacy" {
		t.Fatalf("legacy lesson id = %q, want les_legacy", lessonID)
	}
}

func TestDeleteStudentHidesAndRestoresLessons(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "student-manager.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	ctx := t.Context()
	user := User{
		ID:        "usr_delete_test",
		Name:      "测试老师",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := store.createUser(ctx, user, "unused"); err != nil {
		t.Fatal(err)
	}
	student, err := store.createStudent(ctx, user.ID, Student{
		ID:   "stu_delete_test",
		Name: "待删除学生",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.createLesson(ctx, user.ID, Lesson{
		ID:              "les_delete_test",
		StudentID:       student.ID,
		Date:            "2026-11-11",
		StartTime:       "18:00",
		DurationMinutes: 60,
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := store.deleteStudent(ctx, user.ID, student.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("student was not deleted")
	}
	students, err := store.listStudents(ctx, user.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(students) != 0 {
		t.Fatalf("deleted student still visible: %+v", students)
	}
	exists, err := store.studentBelongsToUser(ctx, user.ID, student.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("deleted student still accepts new lessons")
	}
	lessons, err := store.listLessons(ctx, user.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 0 {
		t.Fatalf("lessons for deleted student still visible: %+v", lessons)
	}

	restored, err := store.createStudent(ctx, user.ID, Student{
		ID:   "stu_replacement_id",
		Name: "待删除学生",
	})
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID != student.ID {
		t.Fatalf("restored student id = %q, want %q", restored.ID, student.ID)
	}
	lessons, err = store.listLessons(ctx, user.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(lessons) != 1 {
		t.Fatalf("restored lesson count = %d, want 1", len(lessons))
	}
}

func tableColumns(store *Store, table string) ([]string, error) {
	rows, err := store.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns := make([]string, 0)
	for rows.Next() {
		var (
			columnID     int
			name         string
			columnType   string
			notNull      int
			defaultValue any
			primaryKey   int
		)
		if err := rows.Scan(&columnID, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}
