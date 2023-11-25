package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestGetCount(t *testing.T) {
	// Assuming count starts at 0
	expected := 0
	actual := getCount()
	if actual != expected {
		t.Errorf("Expected count to be %d, got %d", expected, actual)
	}
}

func TestMessageHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/message", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(messageHandler)

	handler.ServeHTTP(rr, req)

	// Check the status code and response body
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Use a regular expression to match the dynamic response
	expectedPattern := `<li><span class="time">.*</span>: </li>`
	match, _ := regexp.MatchString(expectedPattern, rr.Body.String())
	if !match {
		t.Errorf("handler returned unexpected body: got %v want pattern %v", rr.Body.String(), expectedPattern)
	}
}
