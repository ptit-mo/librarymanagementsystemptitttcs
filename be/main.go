package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	SetLogs()
	router := mux.NewRouter()
	dbx, err := sqlx.Connect("sqlite3", "library.db")
	if err != nil {
		panic(err)
	}
	var bookStore BookStore = NewSQLite3BookStore(dbx)
	var userStore UserStore = NewSQLite3UserStore(dbx)
	var borrowHistoryStore BorrowHistoryStore = NewSQLite3BorrowHistoryStore(dbx)
	var sessionStore SessionStore = NewSQLite3SessionStore(dbx)

	handler := NewHandler(sessionStore, bookStore, userStore, borrowHistoryStore)
	RoutesMux(handler, router)
	SetCors(router)
	Serve(router)
}

func RoutesMux(handler *Handler, r *mux.Router) {
	public := r.PathPrefix("/").Subrouter()
	public.HandleFunc("/login", handler.Login).Methods(http.MethodPost)

	user := r.PathPrefix("/user").Subrouter()
	user.Use(handler.GenerateAuthMiddleware(Borrower))
	user.HandleFunc("/logout", handler.Logout).Methods(http.MethodPost)
	user.HandleFunc("/book/{id}", handler.GetBookDetails).Methods(http.MethodGet)
	user.HandleFunc("/mybooks?offset={offset}&limit={limit}", handler.ListMyBooks).Methods(http.MethodGet)
	user.HandleFunc("/books?offset={offset}&limit={limit}", handler.ListAllBooks).Methods(http.MethodGet)
	user.HandleFunc("/borrowhistory?userid={userid}&offset={offset}&limit={limit}", handler.GetBorrowHistory).Methods(http.MethodGet)
	user.HandleFunc("/user/{id}", handler.GetUserByID).Methods(http.MethodGet)

	librarian := r.PathPrefix("/librarian").Subrouter()
	librarian.Use(handler.GenerateAuthMiddleware(Librarian))
	user.HandleFunc("/book", handler.AddBook).Methods(http.MethodPost)
	user.HandleFunc("/book", handler.UpdateBook).Methods(http.MethodPut)
	user.HandleFunc("/book/{id}", handler.RemoveBook).Methods(http.MethodDelete)
	user.HandleFunc("/user", handler.AddUser).Methods(http.MethodPost)
	user.HandleFunc("/user", handler.UpdateUser).Methods(http.MethodPut)
	user.HandleFunc("/user/{id}", handler.RemoveUser).Methods(http.MethodDelete)
	user.HandleFunc("/users", handler.ListUsers).Methods(http.MethodGet)
	user.HandleFunc("/bookborrow", handler.BorrowBook).Methods(http.MethodPost)
	user.HandleFunc("/bookreturn", handler.ReturnBook).Methods(http.MethodPost)
}
