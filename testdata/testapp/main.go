// Command testapp exercises every registration style go-postman-66 understands.
// It lives under testdata/ so the go tool never builds it.
package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/mux"
	"github.com/labstack/echo/v4"
)

// Lists all users known to the system.
func getUsers(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	mux := http.NewServeMux()
	// Serve the health probe.
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /users", getUsers)
	mux.HandleFunc("DELETE /users/{id}", deleteUser)
	mux.Handle("GET /assets/", http.FileServer(http.Dir("/srv/assets")))

	r := mux.NewRouter()
	r.HandleFunc("/gorilla/orders", createOrder)

	g := gin.New()
	g.GET("/gin/items", listItems)
	g.POST("/gin/items", ginCreateItem)

	c := chi.NewRouter()
	c.Get("/chi/reports", reportsIndex)

	e := echo.New()
	e.GET("/echo/ping", pong)

	f := fiber.New()
	f.Get("/fiber/tasks/:id", taskPage)

	log.Fatal(http.ListenAndServe(":8080", mux))
}

// Removes a user and invalidates their sessions.
func deleteUser(w http.ResponseWriter, _ *http.Request) {}

func createOrder(w http.ResponseWriter, _ *http.Request) {}

func listItems(*gin.Context)                              {}
func ginCreateItem(*gin.Context)                          {}
func reportsIndex(w http.ResponseWriter, r *http.Request) {}
func pong(echo.Context) error                             { return nil }
func taskPage(*fiber.Ctx) error                           { return nil }
