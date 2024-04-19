package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	Admin     string = "admin"
	Librarian        = "librarian"
	Borrower         = "borrower"
)

type User struct {
	ID        int64     `json:"id,omitempty" db:"id"`
	Email     string    `json:"email,omitempty" db:"email"`
	UserName  string    `json:"username,omitempty" db:"username"`
	Password  string    `json:"password,omitempty" db:"password"`
	Type      string    `json:"type,omitempty" db:"type"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type Book struct {
	ID        int64     `json:"id,omitempty" db:"id"`
	Title     string    `json:"title,omitempty" db:"title"`
	Author    string    `json:"author,omitempty" db:"author"`
	Type      string    `json:"type,omitempty" db:"type"`
	CoverUrl  string    `json:"cover" db:"cover"`
	Count     int       `json:"count" db:"count"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type BorrowHistory struct {
	ID         int64     `json:"id,omitempty" db:"id"`
	UserID     int64     `json:"user_id,omitempty" db:"user_id"`
	BookID     int64     `json:"book_id,omitempty" db:"book_id"`
	BorrowedAt time.Time `json:"borrowed_at,omitempty" db:"borrowed_at"`
	Returned   bool      `json:"returned" db:"returned"`
}

type Session struct {
	UserID    int64     `json:"user_id,omitempty" db:"user_id"`
	SessionID string    `json:"session_id,omitempty" db:"session_id"`
	CreatedAt time.Time `json:"created_at" db:"updated_at"`
}

type BookStore interface {
	AddBook(book Book) (ID int64, err error)
	GetBookDetails(ID int64) (Book, error)
	UpdateBook(book Book) error
	RemoveBook(ID int64) error
	ListBooks(lastID, limit int64, order string) ([]Book, error)
}

type BorrowHistoryStore interface {
	BorrowBook(user_id, book_id int64) error
	ReturnBook(id int64) error
	ListAllBorrowHistoryByUserID(userID, lastID, limit int64) ([]GetBorrowHistoryDetailResponse, error)
	ListAllBorrowHistory(lastID, limit int64) ([]GetBorrowHistoryDetailResponse, error)
	GetBorrowHistory(userID, bookID int64) (BorrowHistory, error)
	CountActiveBorrowedBooksByUserID(userID int64) (int64, error)
}

type UserStore interface {
	AddUser(user User) (int64, error)
	RemoveUser(ID int64) error
	GetUserByID(ID int64) (User, error)
	UpdateUser(user User) error
	GetUserByCreds(username, password string) (User, error)
	ListUsers(lastID, limit int64, order string, types []string) ([]User, error)
}

type SessionStore interface {
	CreateSession(session Session) error
	GetUserBySession(sessionID string) (GetSessionResponse, error)
	DeleteSession(sessionID string) error
}

// SQLUserStore implements UserStore interface
type SQLUserStore struct {
	db *sqlx.DB
}

func NewSQLUserStore(db *sqlx.DB) *SQLUserStore {
	return &SQLUserStore{db: db}
}

func (s *SQLUserStore) AddUser(user User) (ID int64, err error) {
	const query = `INSERT INTO users (email, username, password, type) VALUES (:email, :username, :password, :type) RETURNING id`
	namedStmt, err := s.db.PrepareNamed(query)
	if err != nil {
		return 0, err
	}
	defer namedStmt.Close()
	var id int64
	err = namedStmt.Get(&id, user)
	return id, err
}

func (s *SQLUserStore) RemoveUser(ID int64) error {
	const query = `DELETE FROM users WHERE id = $1`
	_, err := s.db.Exec(query, ID)
	return err
}

func (s *SQLUserStore) GetUserByID(ID int64) (User, error) {
	const query = `SELECT id, username, email, type FROM users WHERE id = $1`
	var user User
	err := s.db.Get(&user, query, ID)
	return user, err
}

func (s *SQLUserStore) UpdateUser(user User) error {
	const query = `UPDATE users SET email = :email, username = :username, password = :password, type = :type WHERE id = :id`
	_, err := s.db.NamedExec(query, user)
	return err
}

func (s *SQLUserStore) GetUserByCreds(username, password string) (User, error) {
	const query = `SELECT id, username, email, type FROM users WHERE username = $1 AND password = $2`
	var user User
	err := s.db.Get(&user, query, username, password)
	return user, err
}

func (s *SQLUserStore) ListUsers(lastID, limit int64, order string, types []string) ([]User, error) {
	switch order {
	case "asc":
		return s.listUsersAsc(lastID, limit, types)
	case "desc":
		return s.listUsersDesc(lastID, limit, types)
	default:
		return nil, fmt.Errorf("invalid order: %s", order)
	}
}

func (s *SQLUserStore) listUsersAsc(lastID, limit int64, types []string) ([]User, error) {
	if len(types) == 0 {
		types = []string{Admin, Librarian, Borrower}
	}
	args := []interface{}{lastID, types, limit}
	query := `SELECT id, username, email, type FROM users WHERE id > ? and type in (?) ORDER BY id ASC LIMIT ?`
	if lastID <= 0 {
		args = []interface{}{types, limit}
		query = `SELECT id, username, email, type FROM users WHERE type IN (?) ORDER BY id ASC LIMIT ?`
	}
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, err
	}
	var users []User
	query = s.db.Rebind(query)
	err = s.db.Select(&users, query, args...)
	return users, err
}

func (s *SQLUserStore) listUsersDesc(lastID, limit int64, types []string) ([]User, error) {
	if len(types) == 0 {
		types = []string{Admin, Librarian, Borrower}
	}
	args := []interface{}{lastID, types, limit}
	query := `SELECT id, username, email, type FROM users WHERE id < ? and type in (?) ORDER BY id DESC LIMIT ?`
	if lastID <= 0 {
		args = []interface{}{types, limit}
		query = `SELECT id, username, email, type FROM users WHERE type IN (?) ORDER BY id DESC LIMIT ?`
	}
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, err
	}
	var users []User
	query = s.db.Rebind(query)
	err = s.db.Select(&users, query, args...)
	return users, err

}

// SQLBorrowHistoryStore implements BorrowHistoryStore interface
type SQLBorrowHistoryStore struct {
	db *sqlx.DB
}

func NewSQLBorrowHistoryStore(db *sqlx.DB) *SQLBorrowHistoryStore {
	return &SQLBorrowHistoryStore{db: db}
}

func (s *SQLBorrowHistoryStore) BorrowBook(userID, bookID int64) error {
	tx := s.db.MustBegin()
	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()
	var currentlyBorrowing int64
	err = tx.Get(&currentlyBorrowing, "SELECT count(*) FROM borrow_history WHERE user_id = $1 and book_id = $2 and returned = false", userID, bookID)
	if err != nil {
		return fmt.Errorf("failed to check if user is currently borrowing the book: %v", err)
	}
	if currentlyBorrowing > 0 {
		return fmt.Errorf("user is currently borrowing the book")
	}
	_, err = tx.Exec("INSERT INTO borrow_history (user_id, book_id) VALUES ($1, $2) on conflict (user_id, book_id) do update set borrowed_at = current_timestamp, returned = false", userID, bookID)
	if err != nil {
		return fmt.Errorf("failed to insert borrow history: %v", err)
	}
	_, err = tx.Exec("UPDATE books SET count = count - 1 WHERE id = $1 and count > 0", bookID)
	if err != nil {
		return fmt.Errorf("failed to update book count: %v", err)
	}
	return nil
}

func (s *SQLBorrowHistoryStore) ReturnBook(id int64) error {
	tx := s.db.MustBegin()
	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()
	var bookID int64
	err = tx.Get(&bookID, "UPDATE borrow_history SET returned = true WHERE id = $1 and returned = false returning book_id", id)
	if err != nil {
		return fmt.Errorf("failed to update borrow history: %v", err)
	}
	_, err = tx.Exec("UPDATE books SET count = count + 1 WHERE id = $1", bookID)
	if err != nil {
		return fmt.Errorf("failed to update book count: %v", err)
	}
	return nil
}

type GetBorrowHistoryDetailResponse struct {
	ID          int64     `json:"id" db:"id"`
	UserID      int64     `json:"user_id" db:"user_id"`
	Username    string    `json:"username" db:"username"`
	BookID      int64     `json:"book_id" db:"book_id"`
	BookTitle   string    `json:"title" db:"title"`
	Borrowed_at time.Time `json:"borrowed_at" db:"borrowed_at"`
	Returned    bool      `json:"returned" db:"returned"`
}

func (s *SQLBorrowHistoryStore) CountActiveBorrowedBooksByUserID(userID int64) (int64, error) {
	const query = `SELECT count(*) from borrow_history WHERE user_id = $1 and returned = false`
	var cnt int64
	err := s.db.Get(&cnt, query, userID)
	return cnt, err
}

func (s *SQLBorrowHistoryStore) ListAllBorrowHistoryByUserID(userID, lastID, limit int64) ([]GetBorrowHistoryDetailResponse, error) {
	const query = `SELECT bh.id, username, u.id as user_id, title, b.id as book_id, borrowed_at, returned
	FROM borrow_history bh 
	join users u on bh.user_id = u.id 
	join books b on bh.book_id = b.id 
	WHERE user_id = $1 AND bh.id > $2 ORDER BY id DESC LIMIT $3`
	var bh []GetBorrowHistoryDetailResponse
	err := s.db.Select(&bh, query, userID, lastID, limit)
	return bh, err
}

func (s *SQLBorrowHistoryStore) ListAllBorrowHistory(lastID, limit int64) ([]GetBorrowHistoryDetailResponse, error) {
	const query = `SELECT bh.id, username, u.id as user_id, title, b.id as book_id, borrowed_at, returned
	FROM borrow_history bh 
	join users u on bh.user_id = u.id 
	join books b on bh.book_id = b.id 
	WHERE bh.id > $1 ORDER BY id DESC LIMIT $2`
	var bh []GetBorrowHistoryDetailResponse
	err := s.db.Select(&bh, query, lastID, limit)
	return bh, err
}

func (s *SQLBorrowHistoryStore) GetBorrowHistory(userID, bookID int64) (BorrowHistory, error) {
	const query = `SELECT * FROM borrow_history WHERE user_id = $1 AND book_id = $2`
	var bh BorrowHistory
	err := s.db.Get(&bh, query, userID, bookID)
	return bh, err
}

// SQLSessionStore implements SessionStore interface
type SQLSessionStore struct {
	db *sqlx.DB
}

func NewSQLSessionStore(db *sqlx.DB) *SQLSessionStore {
	return &SQLSessionStore{db: db}
}

func (s *SQLSessionStore) CreateSession(session Session) error {
	const query = `INSERT INTO sessions (user_id, session_id) VALUES (:user_id, :session_id) on conflict(user_id) do update set session_id = excluded.session_id `
	_, err := s.db.NamedExec(query, session)
	if err != nil {
		return err
	}
	return nil
}

type GetSessionResponse struct {
	UserID           int64     `json:"user_id" db:"user_id"`
	UserName         string    `json:"username" db:"username"`
	Email            string    `json:"email" db:"email"`
	UserType         string    `json:"type" db:"type"`
	SessionCreatedAt time.Time `json:"session_created_at" db:"session_created_at"`
}

func (s *SQLSessionStore) GetUserBySession(sessionID string) (GetSessionResponse, error) {
	const query = `SELECT u.id as user_id, u.username, u.email, u.type, s.updated_at as session_created_at  
	FROM users u join sessions s on u.id = s.user_id and session_id = $1`
	var user GetSessionResponse
	err := s.db.Get(&user, query, sessionID)
	return user, err
}

func (s *SQLSessionStore) DeleteSession(sessionID string) error {
	const query = `DELETE FROM sessions WHERE session_id = $1`
	_, err := s.db.Exec(query, sessionID)
	return err
}

// SQLBookStore implements BookStore interface
type SQLBookStore struct {
	db *sqlx.DB
}

func NewSQLBookStore(db *sqlx.DB) *SQLBookStore {
	return &SQLBookStore{db: db}
}

func (s *SQLBookStore) AddBook(book Book) (ID int64, err error) {
	const query = `INSERT INTO books (title, author, type, cover, count) VALUES (:title, :author, :type, :cover, :count) RETURNING id`
	namedStmt, err := s.db.PrepareNamed(query)
	if err != nil {
		return 0, err
	}
	defer namedStmt.Close()
	var id int64
	err = namedStmt.Get(&id, book)
	return id, err
}

func (s *SQLBookStore) GetBookDetails(ID int64) (Book, error) {
	const query = `SELECT * FROM books WHERE id = $1`
	var book Book
	err := s.db.Get(&book, query, ID)
	return book, err
}

func (s *SQLBookStore) UpdateBook(book Book) error {
	const query = `UPDATE books SET title = :title, author = :author, type = :type, cover = :cover, count = :count WHERE id = :id`
	_, err := s.db.NamedExec(query, book)
	return err
}

func (s *SQLBookStore) RemoveBook(ID int64) error {
	const query = `DELETE FROM books WHERE id = $1`
	_, err := s.db.Exec(query, ID)
	return err
}

func (s *SQLBookStore) ListBooks(lastID, limit int64, order string) ([]Book, error) {
	switch order {
	case "asc":
		return s.listBooksAsc(lastID, limit)
	case "desc":
		return s.listBooksDesc(lastID, limit)
	default:
		return nil, fmt.Errorf("invalid order: %s", order)
	}
}

func (s *SQLBookStore) listBooksAsc(lastID, limit int64) ([]Book, error) {
	args := []interface{}{lastID, limit}
	query := `SELECT * FROM books WHERE id > $1 ORDER BY id ASC LIMIT $2`
	if lastID <= 0 {
		args = []interface{}{limit}
		query = `SELECT * FROM books ORDER BY id ASC LIMIT $1`
	}
	var books []Book
	err := s.db.Select(&books, query, args...)
	return books, err
}

func (s *SQLBookStore) listBooksDesc(lastID, limit int64) ([]Book, error) {
	args := []interface{}{lastID, limit}
	query := `SELECT * FROM books WHERE id < $1 ORDER BY id DESC LIMIT $2`
	if lastID <= 0 {
		args = []interface{}{limit}
		query = `SELECT * FROM books ORDER BY id DESC LIMIT $1`
	}
	var books []Book
	err := s.db.Select(&books, query, args...)
	return books, err
}

type ImageStore interface {
	UploadImage(ctx context.Context, image io.Reader, fileSize int64, fileName string) (string, error)
}

type minioImageStore struct {
	*minio.Client
	bucket string
}

func NewMinioImageStore(endpoint, accessKey, secretKey, bucket string, useSSL bool) (ImageStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	return &minioImageStore{Client: client, bucket: bucket}, nil
}

func (s *minioImageStore) UploadImage(ctx context.Context, imageReader io.Reader, fileSize int64, fileName string) (string, error) {
	// calculate hash of the image and use it as the filename to prevent duplicate uploads
	var buf bytes.Buffer
	teeReader := io.TeeReader(imageReader, &buf)
	hash := md5.New()
	io.Copy(hash, teeReader)
	hashSum := hash.Sum(nil)
	hashString := hex.EncodeToString(hashSum)
	user := ctx.Value("user").(GetSessionResponse)
	fileName = fmt.Sprintf("%d__%s__%s__%s", user.UserID, user.UserName, hashString, fileName) // avoid conflict if users upload the same file
	info, err := s.Client.PutObject(ctx, s.bucket, fileName, &buf, fileSize, minio.PutObjectOptions{})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", info.Bucket, info.Key), nil
}

// func (s *minioImageStore) GetImage(ctx context.Context, minioLocation string) (string, error) {
// 	user := ctx.Value("user").(GetSessionResponse)
// 	fileName = fmt.Sprintf("%s__%s__%s", user.UserName, fileName, randomString(10)) // avoid conflict if users upload the same file
// 	info, err := s.Client.PutObject(ctx, s.bucket, "image.jpg", bytes.NewReader(image), int64(len(image)), minio.PutObjectOptions{})
// 	if err != nil {
// 		return "", err
// 	}
// 	return info.Location, nil
// }

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s__%s", time.Now().Format("20060102150405"), string(b))
}
