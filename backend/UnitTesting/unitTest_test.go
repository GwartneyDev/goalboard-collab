package UnitTesting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/benzend/goalboard/pw"
	"github.com/benzend/goalboard/routes"
	"github.com/benzend/goalboard/utils"
	"github.com/stretchr/testify/assert"
)

type MockDb struct {
	driver string
}

func TestRoutes(t *testing.T) {
	// Create a new mock SQL database connection.
	t.Run("UserRegister", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create mock database: %v", err)
		}
		assert.NoError(t, err)
		defer db.Close()
		ctx := context.Background()
		ctxWithValueDB := context.WithValue(ctx, utils.CTX_KEY_DB, db)
		username := "testuser"
		password := "testpass"

		//check that that there are no errors
		hashedPassword, err := pw.HashPassword(password)
		if err != nil {
			t.Fatalf("fajiled to hash passwor %v", err)
		}
		assert.NoError(t, err)

		// Mock the check for existing user
		mock.ExpectExec("SELECT id FROM user_ WHERE username = $1").
			WithArgs(username, hashedPassword).
			WillReturnResult(sqlmock.NewResult(1, 1))
			// Assume no existing user found

		// Expectation for the INSERT operation
		mock.ExpectExec("INSERT INTO user_ (username, password) VALUES ($1, $2)").
			WithArgs(username, hashedPassword).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Create the request with hashed password
		reqBody := []byte(fmt.Sprintf(`{"username": "%s", "password": "%s"}`, username, hashedPassword))
		req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("failed to create HTTP request: %v", err)
		}
		assert.NoError(t, err)

		rr := httptest.NewRecorder()
		routes.Register(ctxWithValueDB, rr, req)

		// Verify that all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())

		// Check for expected status code
		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d but got %d", http.StatusOK, rr.Code)
		}

		// Output the result of the test
		if t.Failed() {
			fmt.Println("TestUserRegister failed")
		} else {
			fmt.Println("TestUserRegister passed")
		}
	})

	t.Run("UserLogin", func(t *testing.T) {
		// Create a mock database connection
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("failed to create mock database: %v", err)
		}
		defer db.Close()

		// Prepare the mock database response for the provided username
		username := "testuser"
		// Ensure that the hashed password matches the one generated by pw.HashPassword
		hashedPassword, err := pw.HashPassword("testpass")
		if err != nil {
			t.Fatalf("failed to hash password: %v", err)
		}
		userId := int64(123)
		mock.ExpectQuery("SELECT password, id FROM user_ WHERE username = ?").
			WithArgs(username).
			WillReturnRows(sqlmock.NewRows([]string{"password", "id"}).AddRow(hashedPassword, userId))

		// Create a request body with user credentials
		requestBody := map[string]string{
			"username": username,
			"password": "testpass", // Use the plaintext password here
		}
		reqBody, err := json.Marshal(requestBody)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}

		// Create a new HTTP request with the request body
		req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Create a mock response recorder
		rr := httptest.NewRecorder()

		// Create a context with the mock database connection
		ctx := context.WithValue(context.Background(), utils.CTX_KEY_DB, db)

		// Call the Login handler function with the mock context, request, and response recorder
		routes.Login(ctx, rr, req)

		// Check the response status code
		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d but got %d", http.StatusOK, rr.Code)
		}

		// Parse the response body and verify its contents
		var responseData routes.LoginReturnData
		err = json.Unmarshal(rr.Body.Bytes(), &responseData)
		if err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		// Verify token and user data
		assert.NotEmpty(t, responseData.Token)
		assert.Equal(t, userId, responseData.User.ID)
		assert.Equal(t, username, responseData.User.Username)

		if t.Failed() {
			fmt.Println("TestLoginHandler failed")
		} else {
			fmt.Println("TestLoginHandler passed")
		}
	})

	t.Run("userLogOut", func(t *testing.T) {
		// Create a new HTTP request (GET /logout)
		req, err := http.NewRequest("GET", "/logout", nil)
		if err != nil {
			t.Fatal("failed to create request:", err)
		}

		// Create a mock response recorder
		rr := httptest.NewRecorder()

		// Create a context (no need for the database in this case)
		ctx := context.Background()

		// Call the Logout handler function with the mock context, response recorder, and request
		routes.Logout(ctx, rr, req)

		// Check the response status code
		if rr.Code != http.StatusSeeOther {
			t.Errorf("expected status code %d but got %d", http.StatusSeeOther, rr.Code)
		}

		// Check if the "jwt_token" cookie is cleared
		cookie := rr.Result().Cookies()[0]
		if cookie.Name != "jwt_token" || cookie.Value != "" || !cookie.Expires.Before(time.Now()) {
			t.Error("jwt_token cookie is not cleared or has incorrect attributes")
		}

		// Check if the response redirects to the login page
		location, err := rr.Result().Location()
		if err != nil {
			t.Fatal("failed to get redirect location:", err)
		}
		if location.Path != "/login" {
			t.Errorf("expected redirect location /login but got %s", location.Path)
		}
	})

}
