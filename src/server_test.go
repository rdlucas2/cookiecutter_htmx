package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetCount(t *testing.T) {
	initDB(true) // Initialize tables in the test database

	// Insert a test count
	_, err := db.Exec("INSERT INTO counts (count) VALUES (5)")
	if err != nil {
		t.Fatal("Failed to insert test count:", err)
	}

	// Test getCount
	result := getCount()
	if result != 5 {
		t.Errorf("Expected count of 5, got %d", result)
	}
}

func TestIncrementCount(t *testing.T) {
	// Test incrementCount
	newCount := incrementCount()
	if newCount != 6 {
		t.Errorf("Expected count of 6, got %d", newCount)
	}

	// Increment again to verify the count increments correctly
	newCount = incrementCount()
	if newCount != 7 {
		t.Errorf("Expected count of 7, got %d", newCount)
	}
}

func TestSaveMessage(t *testing.T) {
	initDB(true)

	// Test saveMessage
	testMessage := "Hello, World!"
	err := saveMessage(testMessage)
	if err != nil {
		t.Errorf("Failed to save message: %v", err)
	}

	// Verify that the message was saved
	var savedMessage string
	err = db.QueryRow("SELECT message FROM messages WHERE message = ?", testMessage).Scan(&savedMessage)
	if err != nil {
		t.Errorf("Failed to retrieve message: %v", err)
	}

	if savedMessage != testMessage {
		t.Errorf("Expected message to be '%s', got '%s'", testMessage, savedMessage)
	}
}

func TestGetLastMessage(t *testing.T) {
	initDB(true)

	// Insert a test message
	testMessage := "Hello, World!"
	_, err := db.Exec("INSERT INTO messages (message) VALUES (?)", testMessage)
	if err != nil {
		t.Fatal("Failed to insert message:", err)
	}

	// Test getLastMessage
	result := getLastMessage()
	if result == "" {
		t.Fatal("getLastMessage returned empty string")
	}

	// Check if the result contains the test message
	if !stringContains(result, testMessage) {
		t.Errorf("Expected getLastMessage to return a string containing '%s', got '%s'", testMessage, result)
	}
}

// Helper function to check if a string contains a substring
func stringContains(str, substr string) bool {
	return strings.Contains(str, substr)
}

func TestRenderTemplate(t *testing.T) {
	// Create a mock ResponseWriter
	w := httptest.NewRecorder()

	// Assuming you have a template file "test_template.html" in the current directory
	// The template file should exist and be valid for this test
	templateFile := "./templates/count.html"
	testData := CountPageData{Count: 6}

	renderTemplate(w, templateFile, testData)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.Status)
	}

	// Optionally, check if response body contains expected content
	respBody := w.Body.String()
	if !strings.Contains(respBody, strconv.Itoa(testData.Count)) {
		t.Errorf("Expected response body to contain %q; got %v", testData.Count, respBody)
	}
}

func TestNotFoundHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(notFoundHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Check the response body
	expected := "404 - Page Not Found"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}

func TestIndexHandler(t *testing.T) {
	// Test for the root path
	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(indexHandler)

	handler.ServeHTTP(rr, req)

	// Check for the correct status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Optionally, check the response body content
	// ...

	// Test for a path that should trigger the notFoundHandler
	req, _ = http.NewRequest("GET", "/invalidpath", nil)
	rr = httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check for the correct status code
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code for not found path: got %v want %v", status, http.StatusNotFound)
	}
}

func TestCountHandler(t *testing.T) {
	initDB(true)

	// Test POST request
	postReq, _ := http.NewRequest("POST", "/count", nil)
	postRR := httptest.NewRecorder()
	postHandler := http.HandlerFunc(countHandler)

	postHandler.ServeHTTP(postRR, postReq)

	// Check for the correct status code and response for POST
	if status := postRR.Code; status != http.StatusOK {
		t.Errorf("POST handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test GET request
	getReq, _ := http.NewRequest("GET", "/count", nil)
	getRR := httptest.NewRecorder()
	getHandler := http.HandlerFunc(countHandler)

	getHandler.ServeHTTP(getRR, getReq)

	// Check for the correct status code and response for GET
	if status := getRR.Code; status != http.StatusOK {
		t.Errorf("GET handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Optionally, add checks for the response body content if needed
}

func TestMessageHandler(t *testing.T) {
	initDB(true)

	// Test POST request
	formData := url.Values{"message": {"Test"}}
	postReq, _ := http.NewRequest("POST", "/message", strings.NewReader(formData.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRR := httptest.NewRecorder()
	postHandler := http.HandlerFunc(messageHandler)

	postHandler.ServeHTTP(postRR, postReq)

	// Check for the correct status code and response for POST
	if status := postRR.Code; status != http.StatusOK {
		t.Errorf("POST handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test GET request
	getReq, _ := http.NewRequest("GET", "/message", nil)
	getRR := httptest.NewRecorder()
	getHandler := http.HandlerFunc(messageHandler)

	getHandler.ServeHTTP(getRR, getReq)

	// Check for the correct status code for GET
	if status := getRR.Code; status != http.StatusOK {
		t.Errorf("GET handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Optionally, add checks for the response body content if needed
}

func TestSendSSECountUpdate(t *testing.T) {
	// Create a mock environment
	mockClients := make(map[chan SSEMessage]struct{})
	mockChan := make(chan SSEMessage, 1) // Buffered channel
	mockClients[mockChan] = struct{}{}

	// Temporarily replace the global sseClients
	originalClients := sseClients
	sseClients = mockClients
	defer func() { sseClients = originalClients }()

	// Call the function
	sendSSECountUpdate(5)

	// Verify that the message was sent
	select {
	case msg := <-mockChan:
		expectedMsg := fmt.Sprintf("%d", 5)
		if msg.Type != "new_count" || msg.Content != expectedMsg {
			t.Errorf("Expected message of type 'new_count' with content '%s', got type '%s' with content '%s'", expectedMsg, msg.Type, msg.Content)
		}
	default:
		t.Error("No message was sent")
	}
}

func TestSendSSETextMessage(t *testing.T) {
	// Create a mock environment
	mockClients := make(map[chan SSEMessage]struct{})
	mockChan := make(chan SSEMessage, 1) // Buffered channel
	mockClients[mockChan] = struct{}{}

	// Temporarily replace the global sseClients
	originalClients := sseClients
	sseClients = mockClients
	defer func() { sseClients = originalClients }()

	// Call the function with a test message
	testMessage := "Test message"
	sendSSETextMessage(testMessage)

	// Verify that the message was sent
	select {
	case msg := <-mockChan:
		if msg.Type != "new_message" || msg.Content != testMessage {
			t.Errorf("Expected message of type 'new_message' with content '%s', got type '%s' with content '%s'", testMessage, msg.Type, msg.Content)
		}
	default:
		t.Error("No message was sent")
	}
}

func TestEventsHandler(t *testing.T) {
	// Create a custom request with a context that times out
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	req := httptest.NewRequest("GET", "/events", nil).WithContext(ctx)
	rr := httptest.NewRecorder()

	handler := http.HandlerFunc(eventsHandler)
	handler.ServeHTTP(rr, req)

	// Check headers
	if contentType := rr.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
	}

	if cacheControl := rr.Header().Get("Cache-Control"); cacheControl != "no-cache" {
		t.Errorf("Expected Cache-Control 'no-cache', got '%s'", cacheControl)
	}

	if connection := rr.Header().Get("Connection"); connection != "keep-alive" {
		t.Errorf("Expected Connection 'keep-alive', got '%s'", connection)
	}

	// Additional checks can be added to test the streaming data
}
