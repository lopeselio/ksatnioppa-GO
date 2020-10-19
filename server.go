package main 

import(
	"encoding/json"
	"net/http"
	"sync"
	"io/ioutil"
	"fmt"
	"time"
	"strings"
	"math/rand"
	"os"

) 

type Meeting struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	ID           string `json:"id"`
}

type meetingHandlers struct {
	sync.Mutex
	store map[string]Meeting 
}

func (h *meetingHandlers) meetings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("method not allowed"))
		return
	}
}

func (h *meetingHandlers) get(w http.ResponseWriter, r *http.Request) {
	meetings := make([]Meeting, len(h.store))
	h.Lock()
	i := 0
	for _, meeting := range h.store {
		meetings[i] = meeting 
		i++
	}
	h.Unlock()
	jsonBytes, err := json.Marshal(meetings)
	if err != nil {
		// TODO
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *meetingHandlers) getMeeting(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.String(), "/")
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[2] == "random" {
		h.getRandomMeeting(w, r)
		return
	}

	h.Lock()
	coaster, ok := h.store[parts[2]]
	h.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonBytes, err := json.Marshal(coaster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *meetingHandlers) getRandomMeeting(w http.ResponseWriter, r *http.Request) {
	ids := make([]string, len(h.store))
	h.Lock()
	i := 0
	for id := range h.store {
		ids[i] = id
		i++
	}
	defer h.Unlock()

	var target string
	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/meetings/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *meetingHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s'", ct)))
		return
	}

	var meeting Meeting
	err = json.Unmarshal(bodyBytes, &meeting)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	meeting.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	h.Lock()
	h.store[meeting.ID] = meeting
	defer h.Unlock()
}

func newMeetingHandlers() *meetingHandlers {
	return &meetingHandlers{
		store: map[string]Meeting{},
	}

}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("required env var ADMIN_PASSWORD not set")
	}

	return &adminPortal{password: password} // return admin portal with password
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - unauthorized"))
		return
	}

	w.Write([]byte("<html><h1>Secret admin portal</h1></html>"))
}

func main() {
	meetingHandlers := newMeetingHandlers()
	http.HandleFunc("/meetings", meetingHandlers.meetings)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}

