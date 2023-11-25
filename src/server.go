package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	count        int
	countMutex   sync.Mutex
	message      string
	messageMutex sync.Mutex
	sseClients   = make(map[chan SSEMessage]struct{})
	sseMutex     sync.Mutex
)

func getCount() int {
	countMutex.Lock()
	defer countMutex.Unlock()
	return count
}

func getMessage() string {
	messageMutex.Lock()
	defer messageMutex.Unlock()
	return message
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
		countMutex.Lock()
		count++
		newCount := count
		countMutex.Unlock()

		sendSSECountUpdate(newCount)
		fmt.Fprintf(w, "Count is now: %v", newCount)

		// data := CountPageData{
		// 	Count: newCount,
		// }
		// renderTemplate(w, "templates/count.html", data)
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
		messageMutex.Lock()
		message = r.FormValue("message")
		messageMutex.Unlock()
		sendSSETextMessage(message)
		fmt.Fprintf(w, "Message received: %s", message)
	case "GET":
		data := MessagePageData{
			Time:    time.Now().UTC().Format(time.RFC3339), // RFC3339 is a standard format that includes timezone info
			Message: getMessage(),
		}
		renderTemplate(w, "templates/message.html", data)
	default:
		// Optional: Handle other HTTP methods or return an error
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/count", countHandler)
	http.HandleFunc("/events", eventsHandler)
	http.HandleFunc("/message", messageHandler)
	log.Println("Listening on port 3000...")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
