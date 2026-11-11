package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"
)

type App struct {
	store   *Store
	limiter *loginLimiter
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var (
	monthPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
	datePattern  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timePattern  = regexp.MustCompile(`^\d{2}:\d{2}$`)
)

func NewApp(dbPath string) (*App, error) {
	store, err := NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	return &App{store: store, limiter: &loginLimiter{attempts: make(map[string][]time.Time)}}, nil
}

func (a *App) Close() error {
	return a.store.Close()
}

func (a *App) Routes(assets fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/auth/status", a.handleAuthStatus)
	mux.HandleFunc("POST /api/auth/setup", a.handleSetup)
	mux.HandleFunc("POST /api/auth/login", a.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", a.handleLogout)

	protected := func(pattern string, handler http.HandlerFunc) {
		mux.Handle(pattern, a.requireAuth(a.sameOrigin(handler)))
	}

	protected("GET /api/students", a.handleListStudents)
	protected("POST /api/students", a.handleCreateStudent)
	protected("PATCH /api/students/{id}", a.handleUpdateStudent)
	protected("DELETE /api/students/{id}", a.handleDeleteStudent)
	protected("GET /api/lessons", a.handleListLessons)
	protected("POST /api/lessons", a.handleCreateLesson)
	protected("PATCH /api/lessons/{id}", a.handleUpdateLesson)
	protected("DELETE /api/lessons/{id}", a.handleDeleteLesson)

	mux.Handle("/", spaHandler(assets))
	return mux
}

func (a *App) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	count, err := a.store.userCount(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取账号状态失败")
		return
	}

	response := map[string]any{
		"initialized":   count > 0,
		"authenticated": false,
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		if user, err := a.sessionUser(r.Context(), cookie.Value); err == nil {
			response["authenticated"] = true
			response["user"] = user
		}
	}
	writeJSON(w, http.StatusOK, response)
}

type setupRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (a *App) handleSetup(w http.ResponseWriter, r *http.Request) {
	var request setupRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		writeError(w, http.StatusBadRequest, "请输入教师姓名")
		return
	}
	if len([]rune(request.Name)) > 40 {
		writeError(w, http.StatusBadRequest, "姓名不能超过 40 个字符")
		return
	}
	if len(request.Password) < 8 {
		writeError(w, http.StatusBadRequest, "密码至少需要 8 位")
		return
	}

	if _, _, err := a.store.findUserByName(r.Context(), request.Name); err == nil {
		writeError(w, http.StatusConflict, "姓名已注册")
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "读取账号失败")
		return
	}

	passwordHash, err := hashPassword(request.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建密码失败")
		return
	}
	user := User{
		ID:        newID("usr"),
		Name:      request.Name,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := a.store.createUser(r.Context(), user, passwordHash); err != nil {
		writeError(w, http.StatusInternalServerError, "创建账号失败")
		return
	}
	token, err := a.createSession(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建登录会话失败")
		return
	}
	a.setSessionCookie(w, r, token)
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !a.limiter.allow(ip) {
		writeError(w, http.StatusTooManyRequests, "尝试次数过多，请 10 分钟后再试")
		return
	}

	var request struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	user, passwordHash, err := a.store.findUserByName(r.Context(), strings.TrimSpace(request.Name))
	if errors.Is(err, sql.ErrNoRows) {
		a.limiter.fail(ip)
		writeError(w, http.StatusNotFound, "该姓名还没有创建账号")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "登录失败，请稍后再试")
		return
	}
	if !verifyPassword(passwordHash, request.Password) {
		a.limiter.fail(ip)
		writeError(w, http.StatusUnauthorized, "密码不正确")
		return
	}
	a.limiter.success(ip)

	token, err := a.createSession(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "创建登录会话失败")
		return
	}
	a.setSessionCookie(w, r, token)
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		_, _ = a.store.db.ExecContext(r.Context(), `DELETE FROM sessions WHERE token_hash = ?`, tokenHash(cookie.Value))
	}
	clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *App) handleListStudents(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	students, err := a.store.listStudents(r.Context(), user.ID, strings.TrimSpace(r.URL.Query().Get("search")))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取学生列表失败")
		return
	}
	writeJSON(w, http.StatusOK, students)
}

func (a *App) handleCreateStudent(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var student Student
	if err := readJSON(r, &student); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	if message := normalizeStudent(&student); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	student.ID = newID("stu")
	student, err := a.store.createStudent(r.Context(), user.ID, student)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "添加学生失败")
		return
	}
	writeJSON(w, http.StatusCreated, student)
}

func (a *App) handleUpdateStudent(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var student Student
	if err := readJSON(r, &student); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	if message := normalizeStudent(&student); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	student.ID = r.PathValue("id")
	ok, err := a.store.updateStudent(r.Context(), user.ID, student.ID, student)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "更新学生资料失败")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "没有找到该学生")
		return
	}
	writeJSON(w, http.StatusOK, student)
}

func (a *App) handleDeleteStudent(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ok, err := a.store.deleteStudent(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "删除学生失败")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "没有找到该学生")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func normalizeStudent(student *Student) string {
	student.Name = strings.TrimSpace(student.Name)
	if student.Name == "" {
		return "请输入学生姓名"
	}
	if len([]rune(student.Name)) > 40 {
		return "学生姓名不能超过 40 个字符"
	}
	return ""
}

func (a *App) handleListLessons(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	month := ""
	if r.URL.Query().Get("all") != "true" {
		var message string
		month, message = normalizeMonth(r.URL.Query().Get("month"))
		if message != "" {
			writeError(w, http.StatusBadRequest, message)
			return
		}
	}
	lessons, err := a.store.listLessons(
		r.Context(), user.ID, month,
		strings.TrimSpace(r.URL.Query().Get("studentId")),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取课程记录失败")
		return
	}
	writeJSON(w, http.StatusOK, lessons)
}

type lessonRequest struct {
	StudentID       string `json:"studentId"`
	Date            string `json:"date"`
	StartTime       string `json:"startTime"`
	DurationMinutes int    `json:"durationMinutes"`
	Status          string `json:"status"`
}

func (a *App) handleCreateLesson(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var request lessonRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	if message := normalizeLessonRequest(&request); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	exists, err := a.store.studentBelongsToUser(r.Context(), user.ID, request.StudentID)
	if err != nil || !exists {
		writeError(w, http.StatusBadRequest, "请选择有效学生")
		return
	}
	lesson := Lesson{
		ID:              newID("les"),
		StudentID:       request.StudentID,
		Date:            request.Date,
		StartTime:       request.StartTime,
		DurationMinutes: request.DurationMinutes,
		Status:          normalizeLessonStatus(request.Status),
	}
	if err := a.store.createLesson(r.Context(), user.ID, lesson); err != nil {
		writeError(w, http.StatusInternalServerError, "保存课程失败")
		return
	}
	saved, err := a.store.getLesson(r.Context(), user.ID, lesson.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取新建课程失败")
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

func (a *App) handleUpdateLesson(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	var request lessonRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "请求内容格式不正确")
		return
	}
	if message := normalizeLessonRequest(&request); message != "" {
		writeError(w, http.StatusBadRequest, message)
		return
	}
	exists, err := a.store.studentBelongsToUser(r.Context(), user.ID, request.StudentID)
	if err != nil || !exists {
		writeError(w, http.StatusBadRequest, "请选择有效学生")
		return
	}
	lesson := Lesson{
		StudentID:       request.StudentID,
		Date:            request.Date,
		StartTime:       request.StartTime,
		DurationMinutes: request.DurationMinutes,
		Status:          normalizeLessonStatus(request.Status),
	}
	ok, err := a.store.updateLesson(r.Context(), user.ID, r.PathValue("id"), lesson)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "更新课程失败")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "没有找到该课程")
		return
	}
	saved, err := a.store.getLesson(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "读取更新后的课程失败")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (a *App) handleDeleteLesson(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	ok, err := a.store.deleteLesson(r.Context(), user.ID, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "删除课程失败")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "没有找到该课程")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func normalizeLessonRequest(request *lessonRequest) string {
	if request.StudentID == "" {
		return "请选择上课学生"
	}
	if !datePattern.MatchString(request.Date) {
		return "请选择有效的上课日期"
	}
	if _, err := time.Parse("2006-01-02", request.Date); err != nil {
		return "上课日期格式不正确"
	}
	if !lessonDateWithinOneMonth(request.Date, shanghaiNow()) {
		return "上课日期最多只能选择未来一个月"
	}
	if !timePattern.MatchString(request.StartTime) {
		return "请选择有效的开始时间"
	}
	if _, err := time.Parse("15:04", request.StartTime); err != nil {
		return "开始时间格式不正确"
	}
	if request.DurationMinutes < 15 || request.DurationMinutes > 240 || request.DurationMinutes%15 != 0 {
		return "课程时长需要是 15 到 240 分钟之间、按 15 分钟递增"
	}
	if request.Status == "" {
		request.Status = "scheduled"
	}
	if request.Status != "scheduled" && request.Status != "attended" {
		return "上课状态不正确"
	}
	return ""
}

func lessonDateWithinOneMonth(date string, now time.Time) bool {
	maxDate := now.AddDate(0, 1, 0).Format("2006-01-02")
	return date <= maxDate
}

func normalizeLessonStatus(status string) string {
	switch status {
	case "attended":
		return status
	default:
		return "scheduled"
	}
}

func normalizeMonth(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return shanghaiNow().Format("2006-01"), ""
	}
	if !monthPattern.MatchString(value) {
		return "", "月份格式不正确"
	}
	if _, err := time.Parse("2006-01", value); err != nil {
		return "", "月份格式不正确"
	}
	return value, ""
}

func shanghaiNow() time.Time {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Now()
	}
	return time.Now().In(location)
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	entries := l.attempts[key]
	filtered := entries[:0]
	for _, timestamp := range entries {
		if timestamp.After(cutoff) {
			filtered = append(filtered, timestamp)
		}
	}
	l.attempts[key] = filtered
	return len(filtered) < 5
}

func (l *loginLimiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[key] = append(l.attempts[key], time.Now())
}

func (l *loginLimiter) success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func newID(prefix string) string {
	buffer := make([]byte, 10)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(buffer)
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func spaHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requested == "" || requested == "." {
			requested = "index.html"
		}
		if _, err := fs.Stat(assets, requested); err != nil {
			content, readErr := fs.ReadFile(assets, "index.html")
			if readErr != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(content)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
