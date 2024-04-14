package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file, make sure you set enough environment according to the .env.sample: %v", err)
	}
	SetLogs()
	router := mux.NewRouter()
	dbx, err := sqlx.Connect("sqlite3", "library.db")
	if err != nil {
		log.Fatalf("sqlx.Connect: %v", err)
	}
	var bookStore BookStore = NewSQLite3BookStore(dbx)
	var userStore UserStore = NewSQLite3UserStore(dbx)
	var borrowHistoryStore BorrowHistoryStore = NewSQLite3BorrowHistoryStore(dbx)
	var sessionStore SessionStore = NewSQLite3SessionStore(dbx)
	imageStore, err := NewMinioImageStore(
		os.Getenv("MINIO_ENDPOINT"),
		os.Getenv("MINIO_ACCESS_KEY"),
		os.Getenv("MINIO_SECRET_KEY"),
		os.Getenv("MINIO_BUCKET"),
		false,
	)
	if err != nil {
		log.Fatalf("NewMinioImageStore: %v", err)
	}
	loginDurationInSecond, err := strconv.ParseInt(os.Getenv("LOGIN_DURATION_IN_SECOND"), 10, 64)
	if err != nil {
		log.Fatalf("empty or invalid setting for env LOGIN_DURATION_IN_SECOND: %s. expect INTEGER", os.Getenv("LOGIN_DURATION_IN_SECOND"))
	}
	handler := NewHandler(sessionStore, bookStore, userStore, borrowHistoryStore, imageStore, loginDurationInSecond)
	RoutesMux(handler, router)
	SetCors(router)
	Serve(router)
}

func RoutesMux(handler *Handler, r *mux.Router) {
	static := http.FileServer(http.Dir("fe"))
	r.PathPrefix("/fe/").Handler(http.StripPrefix("/fe/", static))

	public := r.PathPrefix("/").Subrouter()
	public.HandleFunc("/login", handler.Login).Methods(http.MethodPost)
	public.HandleFunc("/logout", handler.Logout).Methods(http.MethodPost)

	internal := r.PathPrefix("/internal").Subrouter()
	internal.Use(handler.GenerateAuthMiddleware(Borrower))
	internal.HandleFunc("/book/{id}", handler.GetBookDetails).Methods(http.MethodGet)
	internal.HandleFunc("/mybooks", handler.ListMyBooks).Methods(http.MethodGet)
	internal.HandleFunc("/books", handler.ListAllBooks).Methods(http.MethodGet)
	internal.HandleFunc("/borrowhistory", handler.GetBorrowHistory).Methods(http.MethodGet)
	internal.HandleFunc("/user/{id}", handler.GetUserByID).Methods(http.MethodGet)

	admin := r.PathPrefix("/admin").Subrouter()
	admin.Use(handler.GenerateAuthMiddleware(Librarian))
	admin.HandleFunc("/user", handler.AddUser).Methods(http.MethodPost)
	admin.HandleFunc("/user", handler.UpdateUser).Methods(http.MethodPut)
	admin.HandleFunc("/user/{id}", handler.RemoveUser).Methods(http.MethodDelete)
	admin.HandleFunc("/users", handler.ListUsers).Methods(http.MethodGet)
	admin.HandleFunc("/uploadimage", handler.UploadImage).Methods(http.MethodPost)

	librarian := r.PathPrefix("/librarian").Subrouter()
	librarian.Use(handler.GenerateAuthMiddleware(Librarian))
	librarian.HandleFunc("/book", handler.AddBook).Methods(http.MethodPost)
	librarian.HandleFunc("/book", handler.UpdateBook).Methods(http.MethodPut)
	librarian.HandleFunc("/book/{id}", handler.RemoveBook).Methods(http.MethodDelete)
	librarian.HandleFunc("/bookborrow", handler.BorrowBook).Methods(http.MethodPost)
	librarian.HandleFunc("/bookreturn", handler.ReturnBook).Methods(http.MethodPost)
}
