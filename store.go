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
	Count     int       `json:"count,omitempty" db:"count"`
	CreatedAt time.Time `json:"-" db:"created_at"`
	UpdatedAt time.Time `json:"-" db:"updated_at"`
}

type BorrowHistory struct {
	ID         int64     `json:"id,omitempty" db:"id"`
	UserID     int64     `json:"user_id,omitempty" db:"user_id"`
	BookID     int64     `json:"book_id,omitempty" db:"book_id"`
	BorrowedAt time.Time `json:"borrowed_at,omitempty" db:"borrowed_at"`
	ReturnedAt time.Time `json:"returned_at,omitempty" db:"returned_at"`
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
	ListBooks(offset, limit int64) ([]Book, error)
}

type BorrowHistoryStore interface {
	AddBorrowHistory(BorrowHistory) error
	UpdateBorrowHistory(BorrowHistory) error
	ListAllBorrowHistoryByUserID(userID, offset, limit int64) ([]GetBorrowHistoryDetailResponse, error)
	ListAllBorrowHistory(offset, limit int64) ([]GetBorrowHistoryDetailResponse, error)
	GetBorrowHistory(userID, bookID int64) (BorrowHistory, error)
}

type UserStore interface {
	AddUser(user User) (int64, error)
	RemoveUser(ID int64) error
	GetUserByID(ID int64) (User, error)
	UpdateUser(user User) error
	GetUserByCreds(username, password string) (User, error)
	ListUsers(offset, limit int64, types []string) ([]User, error)
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

func (s *SQLUserStore) ListUsers(offset, limit int64, types []string) ([]User, error) {
	var (
		users []User
		err   error
	)
	query := `SELECT id, username, email, type FROM users WHERE id > ? AND type IN (?) LIMIT ?`
	if len(types) == 0 {
		types = []string{Admin, Librarian, Borrower}
	}
	query, args, err := sqlx.In(query, offset, types, limit)
	if err != nil {
		return nil, err
	}
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

func (s *SQLBorrowHistoryStore) AddBorrowHistory(bh BorrowHistory) error {
	const query = `INSERT INTO borrow_history (user_id, book_id) VALUES (:user_id, :book_id)`
	_, err := s.db.NamedExec(query, bh)
	return err
}

func (s *SQLBorrowHistoryStore) UpdateBorrowHistory(bh BorrowHistory) error {
	const query = `UPDATE borrow_history SET user_id = :user_id, book_id = :book_id, returned_at = current_timestamp WHERE id = :id`
	_, err := s.db.NamedExec(query, bh)
	return err
}

type GetBorrowHistoryDetailResponse struct {
	ID          int64     `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	BookTitle   string    `json:"title" db:"title"`
	Borrowed_at time.Time `json:"borrowed_at" db:"borrowed_at"`
	Returned_at time.Time `json:"returned_at" db:"returned_at"`
}

func (s *SQLBorrowHistoryStore) ListAllBorrowHistoryByUserID(userID, offset, limit int64) ([]GetBorrowHistoryDetailResponse, error) {
	const query = `SELECT bh.id, username, title, borrowed_at, returned_at
	FROM borrow_history bh 
	join users u on bh.user_id = u.id 
	join books b on bh.book_id = b.id 
	WHERE user_id = $1 AND bh.id > $2 LIMIT $3`
	var bh []GetBorrowHistoryDetailResponse
	err := s.db.Select(&bh, query, userID, offset, limit)
	return bh, err
}

func (s *SQLBorrowHistoryStore) ListAllBorrowHistory(offset, limit int64) ([]GetBorrowHistoryDetailResponse, error) {
	const query = `SELECT bh.id, username, title, borrowed_at, returned_at
	FROM borrow_history bh 
	join users u on bh.user_id = u.id 
	join books b on bh.book_id = b.id 
	WHERE bh.id > $1 LIMIT $2`
	var bh []GetBorrowHistoryDetailResponse
	err := s.db.Select(&bh, query, offset, limit)
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

func (s *SQLBookStore) ListBooks(offset, limit int64) ([]Book, error) {
	var query = `SELECT * FROM books WHERE id > $1 LIMIT $2`
	var books []Book
	err := s.db.Select(&books, query, offset, limit)
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
