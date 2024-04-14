package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	*BaseHandler
	BookStore
	UserStore
	SessionStore
	BorrowHistoryStore
	LoginDurationInSecond int64
	maxRequestBodySize    int64
	ImageStore
}

func NewHandler(sessionStore SessionStore, book BookStore, user UserStore, bh BorrowHistoryStore, imageStore ImageStore, loginDurationInSecond int64) *Handler {
	return &Handler{
		BookStore:             book,
		UserStore:             user,
		SessionStore:          sessionStore,
		BorrowHistoryStore:    bh,
		BaseHandler:           NewBaseHandler(),
		LoginDurationInSecond: loginDurationInSecond,
		ImageStore:            imageStore,
		maxRequestBodySize:    getMaxRequestBodySize(),
	}
}

const cookieName = "session"

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	SessionID string    `json:"session_id"`
	UserID    int64     `json:"user_id"`
	ExpiredAt time.Time `json:"expired_at"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var logger = logrus.WithFields(logrus.Fields{"route": "/login"})
	var req LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	defer r.Body.Close()
	user, err := h.UserStore.GetUserByCreds(req.Username, req.Password)
	if err != nil {
		logger.WithError(err).Info("get user by creds")
		h.JSONUnauthorized(w, "wrong username/ password")
		return
	}
	sessionID := fmt.Sprintf("session-%d-%d-%s", user.ID, time.Now().Unix(), randomString(10))
	err = h.SessionStore.CreateSession(Session{
		UserID:    user.ID,
		SessionID: sessionID,
	})
	if err != nil {
		logger.WithError(err).Info("create session")
		h.JSONInternalServerError(w, "create session failed")
		return
	}
	expiredAt := time.Now().Add(time.Duration(h.LoginDurationInSecond) * time.Second)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    sessionID,
		Expires:  expiredAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	})
	h.JSONOK(w, LoginResponse{
		SessionID: sessionID,
		UserID:    user.ID,
		ExpiredAt: expiredAt,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var logger = logrus.WithFields(logrus.Fields{"route": "/logout"})
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		logger.WithError(err).Info("get cookie")
		h.JSONBadRequest(w, "get cookie failed")
		return
	}
	err = h.SessionStore.DeleteSession(cookie.Value)
	if err != nil {
		logger.WithError(err).Info("logout failed")
		h.JSONBadRequest(w, "logout failed")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:   cookieName,
		Value:  "deleted",
		MaxAge: -1,
	})
	h.JSONOK(w, struct{}{})
}

func (h *Handler) GenerateAuthMiddleware(userType string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return h.AuthMiddleware(next, userType)
	}
}

func (h *Handler) AuthMiddleware(next http.Handler, userType string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var logger = logrus.WithFields(logrus.Fields{"route": r.RequestURI})
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			logger.WithError(err).Info("get cookie")
			h.JSONUnauthorized(w, "unauthorized")
			return
		}
		usersession, err := h.SessionStore.GetUserBySession(cookie.Value)
		if err != nil {
			logger.WithError(err).Info("get user by session")
			h.JSONUnauthorized(w, "session expired")
			return
		}
		if usersession.SessionCreatedAt.Add(time.Duration(h.LoginDurationInSecond) * time.Second).Before(time.Now()) {
			logger.Info("session expired")
			http.SetCookie(w, &http.Cookie{
				Name:   cookieName,
				Value:  "deleted",
				MaxAge: -1,
			})
			h.JSONUnauthorized(w, "session expired")
			return
		}
		if usersession.UserType == Librarian && userType == Admin ||
			usersession.UserType == Borrower && userType != Borrower {
			logger.Info("unauthorized")
			h.JSONUnauthorized(w, "unauthorized")
			return

		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, "user", usersession)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (h *Handler) AddBook(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req Book
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	defer r.Body.Close()
	id, iErr := h.BookStore.AddBook(req)
	if iErr != nil {
		logger.WithError(iErr).Info("add book failed")
		h.JSONInternalServerError(w, "add book failed")
		return
	}
	book, bErr := h.BookStore.GetBookDetails(id)
	if bErr != nil {
		logger.WithError(bErr).Info("get book details failed")
		h.JSONInternalServerError(w, "get book details failed")
		return
	}
	h.JSONOK(w, book)
}

func (h *Handler) UpdateBook(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req Book
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	defer r.Body.Close()
	iErr := h.BookStore.UpdateBook(req)
	if iErr != nil {
		logger.WithError(iErr).Info("update book failed")
		h.JSONInternalServerError(w, "add book failed")
		return
	}
	book, bErr := h.BookStore.GetBookDetails(req.ID)
	if bErr != nil {
		logger.WithError(bErr).Info("get book details failed")
		h.JSONInternalServerError(w, "get book details failed")
		return
	}
	h.JSONOK(w, book)
}

func (h *Handler) RemoveBook(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	bookID := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(bookID, 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse book id")
		h.JSONBadRequest(w, "parse book id failed")
		return
	}
	err = h.BookStore.RemoveBook(id)
	if err != nil {
		logger.WithError(err).Info("remove book failed")
		h.JSONInternalServerError(w, "remove book failed")
		return
	}
	h.JSONOK(w, map[string]int64{"id": id})
}

func (h *Handler) GetBookDetails(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	bookID := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(bookID, 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse book id")
		h.JSONBadRequest(w, "parse book id failed")
		return
	}
	book, err := h.BookStore.GetBookDetails(id)
	if err != nil {
		logger.WithError(err).Info("get book failed")
		h.JSONInternalServerError(w, "remove book failed")
		return
	}
	h.JSONOK(w, book)
}

func (h *Handler) ListMyBooks(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	user := r.Context().Value("user").(GetSessionResponse)
	offset, limit := parseOffsetLimit(r, logger)
	books, err := h.BorrowHistoryStore.ListAllBorrowHistoryByUserID(user.UserID, offset, limit)
	if err != nil {
		logger.WithError(err).Info("list borrowing list failed")
		h.JSONInternalServerError(w, "list borrowing list failed")
		return
	}
	h.JSONOK(w, books)
}

func (h *Handler) ListAllBooks(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	offset, limit := parseOffsetLimit(r, logger)
	books, err := h.BookStore.ListBooks(offset, limit)
	if err != nil {
		logger.WithError(err).Info("list books failed")
		h.JSONInternalServerError(w, "list books failed")
		return
	}
	h.JSONOK(w, books)
}

func parseOffsetLimit(r *http.Request, logger *logrus.Entry) (int64, int64) {
	offset, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if err != nil {
		logger.Debugf("failed to parse offset, use default 0 offset")
		offset = 0
	}
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil {
		logger.Debugf("failed to parse offset, use default 10 limit")
		limit = 10
	}
	return offset, limit
}

func (h *Handler) authorizeUserOperations(req User,
	logger *logrus.Entry,
	operation string,
	w http.ResponseWriter,
	r *http.Request) error {
	user := r.Context().Value("user").(GetSessionResponse)
	if user.UserType == Borrower {
		h.JSONUnauthorized(w, fmt.Sprintf("borrower can't %s another user", operation))
		return fmt.Errorf("unauthorized")
	}
	if user.UserType != Admin && (req.Type == Librarian || req.Type == Admin) {
		logger.Info("unauthorized")
		h.JSONUnauthorized(w, fmt.Sprintf("only admin can %s another admin or librarian", operation))
		return fmt.Errorf("unauthorized")
	}
	return nil
}

func (h *Handler) AddUser(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req User
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	defer r.Body.Close()
	if err := h.authorizeUserOperations(req, logger, "add", w, r); err != nil {
		logger.Info("unauthorized")
		return
	}
	id, iErr := h.UserStore.AddUser(req)
	if iErr != nil {
		logger.WithError(iErr).Info("add user failed")
		h.JSONInternalServerError(w, "add user failed")
		return
	}
	user, bErr := h.UserStore.GetUserByID(id)
	if bErr != nil {
		logger.WithError(bErr).Info("get user details failed")
		h.JSONInternalServerError(w, "get user details failed")
		return
	}
	h.JSONOK(w, user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req User
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	defer r.Body.Close()
	if err := h.authorizeUserOperations(req, logger, "update", w, r); err != nil {
		logger.Info("unauthorized")
		return
	}
	iErr := h.UserStore.UpdateUser(req)
	if iErr != nil {
		logger.WithError(iErr).Info("update user failed")
		h.JSONInternalServerError(w, "add user failed")
		return
	}
	user, bErr := h.UserStore.GetUserByID(req.ID)
	if bErr != nil {
		logger.WithError(bErr).Info("get user details failed")
		h.JSONInternalServerError(w, "get user details failed")
		return
	}
	h.JSONOK(w, user)
}

func (h *Handler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	userID := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse user id")
		h.JSONBadRequest(w, "parse user id failed")
		return
	}
	user, err := h.UserStore.GetUserByID(id)
	if err != nil {
		logger.WithError(err).Info("get user details failed")
		h.JSONInternalServerError(w, "get user details failed")
		return
	}
	if err := h.authorizeUserOperations(user, logger, "remove", w, r); err != nil {
		logger.Info("unauthorized")
		return
	}
	err = h.UserStore.RemoveUser(id)
	if err != nil {
		logger.WithError(err).Info("remove user failed")
		h.JSONInternalServerError(w, "remove user failed")
		return
	}
	h.JSONOK(w, map[string]int64{"id": id})
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	offset, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse offset")
		h.JSONBadRequest(w, "parse offset failed")
		return
	}
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse limit")
		h.JSONBadRequest(w, "parse limit failed")
		return
	}
	user := r.Context().Value("user").(GetSessionResponse)
	if user.UserType == Borrower {
		logger.Info("unauthorized")
		h.JSONUnauthorized(w, "borrower can't see other users")
		return
	}
	var types []string = []string{Librarian, Borrower}
	if user.UserType == Admin {
		types = []string{Admin, Librarian, Borrower} // only admin can see all users
	}
	users, err := h.UserStore.ListUsers(offset, limit, types)
	if err != nil {
		logger.WithError(err).Info("list users failed")
		h.JSONInternalServerError(w, "list users failed")
		return
	}
	h.JSONOK(w, users)
}

func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	userID := mux.Vars(r)["id"]
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse user id")
		h.JSONBadRequest(w, "parse user id failed")
		return
	}
	requestor := r.Context().Value("user").(GetSessionResponse)
	if requestor.UserType == Borrower && requestor.UserID != id {
		logger.Info("unauthorized")
		h.JSONUnauthorized(w, "borrower can't see other users")
		return
	}
	user, err := h.UserStore.GetUserByID(id)
	if err != nil {
		logger.WithError(err).Info("get user details failed")
		h.JSONInternalServerError(w, "get user details failed")
		return
	}
	if requestor.UserType == Librarian && user.Type == Admin {
		logger.Info("user does not exist")
		h.JSONNotFound(w, "user does not exist")
		return
	}
	h.JSONOK(w, user)
}

func (h *Handler) GetBorrowHistory(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	offset, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse offset")
		h.JSONBadRequest(w, "parse offset failed")
		return
	}
	limit, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
	if err != nil {
		logger.WithError(err).Info("failed to parse limit")
		h.JSONBadRequest(w, "parse limit failed")
		return
	}
	requestor := r.Context().Value("user").(GetSessionResponse)
	requestedUserIDStr := r.URL.Query().Get("userid")
	requestedUserID, err := strconv.ParseInt(requestedUserIDStr, 10, 64)
	if err != nil && requestor.UserType == Borrower {
		requestedUserID = requestor.UserID
	}
	if requestor.UserType == Borrower && requestedUserID != requestor.UserID {
		logger.Info("unauthorized")
		h.JSONUnauthorized(w, "borrower can't see other users' borrow history")
		return
	}
	var borrowHistory []GetBorrowHistoryDetailResponse
	if requestedUserID != requestor.UserID { // admin query specific users
		borrowHistory, err = h.BorrowHistoryStore.ListAllBorrowHistory(offset, limit)
	} else { // admin query all users
		borrowHistory, err = h.BorrowHistoryStore.ListAllBorrowHistoryByUserID(requestedUserID, offset, limit)
	}
	if err != nil {
		logger.WithError(err).Info("list borrow history failed")
		h.JSONInternalServerError(w, "list borrow history failed")
		return
	}
	h.JSONOK(w, borrowHistory)
}

func (h *Handler) BorrowBook(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req BorrowHistory
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	book, err := h.BookStore.GetBookDetails(req.BookID)
	if err != nil {
		logger.WithError(err).Info("get book details failed")
		h.JSONInternalServerError(w, "get book details failed")
		return
	}
	if book.Count == 0 {
		logger.Info("book not available")
		h.JSONBadRequest(w, "book not available")
		return
	}
	user, err := h.UserStore.GetUserByID(req.UserID)
	if err != nil {
		logger.WithError(err).Info("get user details failed")
		h.JSONInternalServerError(w, "get user details failed")
		return
	}
	borrowHistory := BorrowHistory{
		UserID:     user.ID,
		BookID:     book.ID,
		BorrowedAt: time.Now(),
	}
	err = h.BorrowHistoryStore.AddBorrowHistory(borrowHistory)
	if err != nil {
		logger.WithError(err).Info("borrow book failed")
		h.JSONInternalServerError(w, "borrow book failed")
		return
	}
	h.JSONOK(w, borrowHistory)
}

func (h *Handler) ReturnBook(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	var req BorrowHistory
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		logger.WithError(err).Info("failed to decode request body")
		h.JSONBadRequest(w, "decode json failed")
		return
	}
	if req.BookID == 0 || req.UserID == 0 {
		logger.Info("invalid request")
		h.JSONBadRequest(w, "book id and user id are required")
		return
	}
	borrowHistory, err := h.BorrowHistoryStore.GetBorrowHistory(req.UserID, req.BookID)
	if err != nil {
		logger.WithError(err).Info("get borrow history failed")
		h.JSONInternalServerError(w, "get borrow history failed")
		return
	}
	if borrowHistory.ReturnedAt != time.Unix(0, 0) {
		logger.Info("book already returned")
		h.JSONBadRequest(w, "book already returned")
		return
	}
	book, err := h.BookStore.GetBookDetails(req.BookID)
	if err != nil {
		logger.WithError(err).Info("get book details failed")
		h.JSONInternalServerError(w, "get book details failed")
		return
	}
	book.Count++
	err = h.BookStore.UpdateBook(book)
	if err != nil {
		logger.WithError(err).Info("update book failed")
		h.JSONInternalServerError(w, "update book failed")
		return
	}
	borrowHistory.ReturnedAt = time.Now()
	err = h.BorrowHistoryStore.UpdateBorrowHistory(borrowHistory)
	if err != nil {
		logger.WithError(err).Info("return book failed")
		h.JSONInternalServerError(w, "return book failed")
		return
	}
	h.JSONOK(w, borrowHistory)
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	logger := logrus.WithFields(logrus.Fields{"route": r.RequestURI})
	err := r.ParseMultipartForm(h.maxRequestBodySize)
	file, handler, err := r.FormFile("file")
	if err != nil {
		logger.WithError(err).Info("retrieving file")
		h.JSONInternalServerError(w, "retrieving file failed")
		return
	}
	defer file.Close()
	filePath, err := h.ImageStore.UploadImage(r.Context(), file, handler.Size, handler.Filename)
	if err != nil {
		logger.WithError(err).Info("upload image failed")
		h.JSONInternalServerError(w, "upload image failed")
		return
	}
	h.JSONOK(w, map[string]string{"path": fmt.Sprintf("http://%s/%s", os.Getenv("MINIO_ENDPOINT"), filePath)})
}
