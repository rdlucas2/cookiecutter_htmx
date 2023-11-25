package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var (
	countMutex   sync.Mutex
	messageMutex sync.Mutex
	sseClients   = make(map[chan SSEMessage]struct{})
	sseMutex     sync.Mutex
	db           *sql.DB
)

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "file:messages.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}

	createCountTable := `
    CREATE TABLE IF NOT EXISTS counts (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        count INTEGER NOT NULL
    );`
	_, err = db.Exec(createCountTable)
	if err != nil {
		log.Fatal(err)
	}

	createMessageTable := `
    CREATE TABLE IF NOT EXISTS messages (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
        message TEXT
    );`
	_, err = db.Exec(createMessageTable)
	if err != nil {
		log.Fatal(err)
	}
}

func getCount() int {
	var count int
	row := db.QueryRow("SELECT count FROM counts ORDER BY id DESC LIMIT 1")
	err := row.Scan(&count)
	if err != nil {
		return 0 // Default to 0 if no count is found
	}
	return count
}

func incrementCount() int {
	countMutex.Lock()
	defer countMutex.Unlock()
	newCount := getCount() + 1
	_, err := db.Exec("INSERT INTO counts (count) VALUES (?)", newCount)
	if err != nil {
		log.Fatal(err)
	}
	return newCount
}

func saveMessage(message string) error {
	messageMutex.Lock()
	defer messageMutex.Unlock()

	_, err := db.Exec("INSERT INTO messages (message) VALUES (?)", message)
	if err != nil {
		return err
	}
	return nil
}

func getLastMessage() string {
	var message string
	var timestamp time.Time
	row := db.QueryRow("SELECT message, timestamp FROM messages ORDER BY id DESC LIMIT 1")
	err := row.Scan(&message, &timestamp)
	if err != nil {
		return "" // Return empty if no message is found
	}
	return fmt.Sprintf("%s: %s", timestamp.Format(time.RFC3339), message)
}

type SSEMessage struct {
	Type    string // "new_count" or "new_message"
	Content string
}

type BasePageData struct {
	Title       string
	Header      string
	Description string
}

type CountPageData struct {
	Count int
}

type MessagePageData struct {
	Time    string
	Message string
}

type IndexPageData struct {
	BasePageData
	CountPageData
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - Page Not Found"))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFoundHandler(w, r)
		return
	}
	base := BasePageData{
		Title:       "Home",
		Header:      "Welcome to My Website!",
		Description: "The homepage of the website of Ryan Lucas",
	}
	count := CountPageData{
		Count: getCount(),
	}
	data := IndexPageData{
		BasePageData:  base,
		CountPageData: count,
	}
	renderTemplate(w, "templates/index.html", data)
}

func countHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		newCount := incrementCount()
		sendSSECountUpdate(newCount)
		fmt.Fprintf(w, "Count is now: %v", newCount)
	case "GET":
		data := CountPageData{
			Count: getCount(),
		}
		renderTemplate(w, "templates/count.html", data)
	default:
		// Optional: Handle other HTTP methods or return an error
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		message := r.FormValue("message")
		err := saveMessage(message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendSSETextMessage(getLastMessage())
		fmt.Fprintf(w, "Message received: %s", message)

	case "GET":
		data := MessagePageData{
			Time:    time.Now().UTC().Format(time.RFC3339),
			Message: getLastMessage(),
		}
		renderTemplate(w, "templates/message.html", data)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func sendSSECountUpdate(newCount int) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	msg := SSEMessage{Type: "new_count", Content: fmt.Sprintf("%d", newCount)}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func sendSSETextMessage(newMessage string) {
	sseMutex.Lock()
	defer sseMutex.Unlock()
	msg := SSEMessage{Type: "new_message", Content: newMessage}
	for clientChan := range sseClients {
		select {
		case clientChan <- msg:
		default:
		}
	}
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan SSEMessage)
	sseMutex.Lock()
	sseClients[clientChan] = struct{}{}
	sseMutex.Unlock()

	defer func() {
		sseMutex.Lock()
		delete(sseClients, clientChan)
		sseMutex.Unlock()
		close(clientChan)
	}()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-clientChan:
			if msg.Type == "new_count" {
				fmt.Fprintf(w, "event: new_count\ndata: %s\n\n", msg.Content)
			} else if msg.Type == "new_message" {
				fmt.Fprintf(w, "event: new_message\ndata: %s\n\n", msg.Content)
			}
			w.(http.Flusher).Flush()
		}
	}
}

func main() {
	initDB()
	defer db.Close()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/count", countHandler)
	http.HandleFunc("/events", eventsHandler)
	http.HandleFunc("/message", messageHandler)
	log.Println("Listening on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
