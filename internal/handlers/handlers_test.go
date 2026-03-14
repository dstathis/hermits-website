package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Println("TEST_DATABASE_URL not set, skipping handler tests")
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open test database: %v", err)
	}

	for i := 0; i < 30; i++ {
		if err := testDB.Ping(); err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := testDB.Ping(); err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	code := m.Run()
	testDB.Close()
	os.Exit(code)
}

func cleanTable(t *testing.T, table string) {
	t.Helper()
	_, err := testDB.Exec(fmt.Sprintf("DELETE FROM %s", table))
	if err != nil {
		t.Fatalf("failed to clean %s: %v", table, err)
	}
}

func cleanAll(t *testing.T) {
	t.Helper()
	cleanTable(t, "sessions")
	cleanTable(t, "admin_users")
	cleanTable(t, "subscribers")
	cleanTable(t, "events")
}

// templateDir returns the path to the templates directory.
func templateDir() string {
	candidates := []string{
		filepath.Join("..", "..", "templates"),
		"templates",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "layout.html")); err == nil {
			return c
		}
	}
	return filepath.Join("..", "..", "templates")
}

var testFuncMap = template.FuncMap{
	"formatDate": func(t time.Time, layout string) string {
		athens, _ := time.LoadLocation("Europe/Athens")
		return t.In(athens).Format(layout)
	},
}

func parsePage(files ...string) *template.Template {
	dir := templateDir()
	layoutFile := filepath.Join(dir, "layout.html")
	paths := []string{layoutFile}
	for _, f := range files {
		paths = append(paths, filepath.Join(dir, f))
	}
	t, err := template.New(filepath.Base(layoutFile)).Funcs(testFuncMap).ParseFiles(paths...)
	if err != nil {
		log.Fatalf("Failed to parse template %v: %v", files, err)
	}
	return t
}
