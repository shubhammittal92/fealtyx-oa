package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/mux"
)

var (
	students = make(map[int]Student)
	mu       sync.Mutex
)

// Student struct to hold student data
type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

func main() {
	router := mux.NewRouter()

	// Register routes
	router.HandleFunc("/students", createStudent).Methods("POST")
	router.HandleFunc("/students", getAllStudents).Methods("GET")
	router.HandleFunc("/students/{id}", getStudentByID).Methods("GET")
	router.HandleFunc("/students/{id}", updateStudent).Methods("PUT")
	router.HandleFunc("/students/{id}", deleteStudent).Methods("DELETE")
	router.HandleFunc("/students/{id}/summary", generateStudentSummary).Methods("GET")

	// Start the server
	log.Println("Server is listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8081", router))
}

// createStudent handles POST /students to create a new student
func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Basic validation
	if student.Name == "" || student.Age <= 0 || student.Email == "" {
		http.Error(w, "Invalid student data", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()
	student.ID = len(students) + 1
	students[student.ID] = student

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(student)
}

// getAllStudents handles GET /students to fetch all students
func getAllStudents(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	var allStudents []Student
	for _, student := range students {
		allStudents = append(allStudents, student)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allStudents)
}

// getStudentByID handles GET /students/{id} to fetch a student by ID
func getStudentByID(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromURL(r.URL.Path)

	mu.Lock()
	defer mu.Unlock()

	student, exists := students[id]
	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// updateStudent handles PUT /students/{id} to update a student by ID
func updateStudent(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromURL(r.URL.Path)

	var updatedStudent Student
	if err := json.NewDecoder(r.Body).Decode(&updatedStudent); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	student, exists := students[id]
	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Update fields
	if updatedStudent.Name != "" {
		student.Name = updatedStudent.Name
	}
	if updatedStudent.Age > 0 {
		student.Age = updatedStudent.Age
	}
	if updatedStudent.Email != "" {
		student.Email = updatedStudent.Email
	}

	students[id] = student

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student)
}

// deleteStudent handles DELETE /students/{id} to delete a student by ID
func deleteStudent(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromURL(r.URL.Path)

	mu.Lock()
	defer mu.Unlock()

	_, exists := students[id]
	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	delete(students, id)

	w.WriteHeader(http.StatusNoContent)
}

// generateStudentSummary calls the Ollama API (Llama2 model) to generate a summary for a student
func generateStudentSummary(w http.ResponseWriter, r *http.Request) {
	id := extractIDFromURL(r.URL.Path)

	mu.Lock()
	defer mu.Unlock()

	student, exists := students[id]
	if !exists {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Construct the input for Ollama with Llama2 model
	summaryRequest := map[string]interface{}{
		"input": fmt.Sprintf("Generate a detailed summary for the following student: Name: %s, Age: %d, Email: %s", student.Name, student.Age, student.Email),
		"model": "llama2", // Specifying Llama2 model
	}

	// Marshal the request to JSON
	summaryRequestJSON, _ := json.Marshal(summaryRequest)

	// Send request to Ollama's localhost server
	resp, err := http.Post("http://localhost:11411/v1/chat/completions", "application/json", bytes.NewBuffer(summaryRequestJSON))
	if err != nil {
		http.Error(w, "Error generating summary", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response from Ollama
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading Ollama response", http.StatusInternalServerError)
		return
	}

	// Respond with the summary
	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// extractIDFromURL extracts student ID from the URL
func extractIDFromURL(url string) int {
	idStr := url[len("/students/"):]
	id, _ := strconv.Atoi(idStr)
	return id
}
