package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileUpload(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	tmp, _ := os.CreateTemp("", "upload*.txt")
	tmp.WriteString("hello world")
	tmp.Seek(0, 0)

	part, _ := writer.CreateFormFile("file", "hello.txt")
	io.Copy(part, tmp)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	TestApp.Router().ServeHTTP(rec, req)

	assert.Contains(t, []int{201, 400, 401}, rec.Code)
}

func TestFileUpload_MissingFile(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	TestApp.Router().ServeHTTP(rec, req)

	assert.Equal(t, 400, rec.Code)
}

func TestFileUpload_PasswordTooShort(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	tmp, _ := os.CreateTemp("", "short_pass*.txt")
	tmp.WriteString("Dau Minh Khoi")
	tmp.Seek(0, 0)

	part, _ := writer.CreateFormFile("file", "short_pass.txt")
	io.Copy(part, tmp)
	writer.WriteField("password", "12345")
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	TestApp.Router().ServeHTTP(rec, req)

	assert.Equal(t, 400, rec.Code)
}

func TestFileUpload_InvalidDateRange(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	tmp, _ := os.CreateTemp("", "date_test*.txt")
	tmp.WriteString("Dau Minh Khoi")
	tmp.Seek(0, 0)

	part, _ := writer.CreateFormFile("file", "date_test.txt")
	io.Copy(part, tmp)
	now := time.Now()
	future := now.Add(24 * time.Hour)
	writer.WriteField("availableFrom", future.Format(time.RFC3339))
	writer.WriteField("availableTo", now.Format(time.RFC3339))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	TestApp.Router().ServeHTTP(rec, req)

	assert.Equal(t, 400, rec.Code)
}

func TestFileUpload_MissingAuth(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	tmp, _ := os.CreateTemp("", "secret*.txt")
	tmp.WriteString("Dau Minh Khoi")
	tmp.Seek(0, 0)

	part, _ := writer.CreateFormFile("file", "secret.txt")
	io.Copy(part, tmp)
	writer.WriteField("password", "123456")
	writer.WriteField("isPublic", "false")
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/files/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	TestApp.Router().ServeHTTP(rec, req)

	assert.Equal(t, 401, rec.Code)

}

func TestFileUpload_SimpleFlow(t *testing.T) {
	username := fmt.Sprintf("testuser_%d", time.Now().UnixNano())
	email := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
	password := "123456789"
	var token string

	// Register New Account
	t.Run("Register New Account", func(t *testing.T) {
		body := fmt.Sprintf(`{
            "username": "%s",
            "email": "%s",
            "password": "%s"
        }`, username, email, password)

		req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		TestApp.Router().ServeHTTP(rec, req)

		assert.Equal(t, 200, rec.Code)
	})

	// Login To New Account
	t.Run("Login To New Account", func(t *testing.T) {
		body := fmt.Sprintf(`{
            "email": "%s",
            "password": "%s"
        }`, email, password)

		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		TestApp.Router().ServeHTTP(rec, req)

		assert.Equal(t, 200, rec.Code)

		jsonResp := ParseJSON(t, rec)
		assert.NotEmpty(t, jsonResp["accessToken"])

		token = jsonResp["accessToken"].(string)
	})

	// Upload File
	t.Run("Upload File", func(t *testing.T) {
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		tmp, _ := os.CreateTemp("", "private*.txt")
		tmp.WriteString("Dau Minh Khoi")
		tmp.Seek(0, 0)
		defer os.Remove(tmp.Name())

		part, _ := writer.CreateFormFile("file", "private.txt")
		io.Copy(part, tmp)
		writer.WriteField("password", "123456")
		writer.WriteField("isPublic", "false")
		writer.Close()

		req := httptest.NewRequest("POST", "/api/files/upload", &buf)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+token)

		rec := httptest.NewRecorder()
		TestApp.Router().ServeHTTP(rec, req)

		assert.Equal(t, 201, rec.Code)
	})
}
