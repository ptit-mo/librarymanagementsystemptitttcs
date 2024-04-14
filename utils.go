package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

type BaseHandler struct {
}

func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

func (h *BaseHandler) BytesResponse(w http.ResponseWriter, contentType string, statusCode int, body []byte) {
	var logger = logrus.WithFields(nil)
	w.Header().Add("Content-Type", contentType)
	w.WriteHeader(statusCode)
	if _, err := w.Write(body); err != nil {
		logger.WithError(err).Error("write body bytes")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *BaseHandler) JSON(w http.ResponseWriter, statusCode int, body interface{}) {
	const contentTypeHeader = `application/json`
	var logger = logrus.WithFields(nil)

	jsonBytes, err := json.Marshal(body)
	if err != nil {
		logger.WithError(err).Error("encode body to json bytes: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.BytesResponse(w, contentTypeHeader, statusCode, jsonBytes)
}

func (h *BaseHandler) JSONOK(w http.ResponseWriter, body interface{}) {
	h.JSON(w, http.StatusOK, body)
}

func (h *BaseHandler) JSONAccepted(w http.ResponseWriter, body interface{}) {
	h.JSON(w, http.StatusAccepted, body)
}

func (h *BaseHandler) JSONCreated(w http.ResponseWriter, body interface{}) {
	h.JSON(w, http.StatusCreated, body)
}

func (h *BaseHandler) JSONBadRequest(w http.ResponseWriter, msg string) {
	h.JSON(w, http.StatusBadRequest, NewBaseResponse(msg))
}

func (h *BaseHandler) JSONUnauthorized(w http.ResponseWriter, msg string) {
	h.JSON(w, http.StatusUnauthorized, NewBaseResponse(msg))
}

func (h *BaseHandler) JSONNotFound(w http.ResponseWriter, msg string) {
	h.JSON(w, http.StatusNotFound, NewBaseResponse(msg))
}

func (h *BaseHandler) JSONStatusConflict(w http.ResponseWriter, msg string) {
	h.JSON(w, http.StatusConflict, NewBaseResponse(msg))
}

func (h *BaseHandler) JSONInternalServerError(w http.ResponseWriter, msg string) {
	h.JSON(w, http.StatusInternalServerError, NewBaseResponse(msg))
}

func (h *BaseHandler) JSONGenericInternalServerError(w http.ResponseWriter) {
	h.JSON(w, http.StatusInternalServerError, NewBaseResponse("Something went wrong, try again later"))
}

func (h *BaseHandler) JSONTooManyRequests(w http.ResponseWriter) {
	h.JSON(w, http.StatusTooManyRequests, NewBaseResponse("Too many requests"))
}

func (h *BaseHandler) TextOk(w http.ResponseWriter, content string) {
	h.BytesResponse(w, "text/plain", http.StatusOK, []byte(content))
}

type BaseResponse struct {
	Message string `json:"message,omitempty"`
}

func NewBaseResponse(msg string) *BaseResponse {
	return &BaseResponse{Message: msg}
}

func SetCors(r *mux.Router) {
	r.Use(cors.New(cors.Options{
		AllowedOrigins:     []string{"https://*", "http://*"},
		AllowedMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowedHeaders:     []string{"Origin", "Accept", "Content-Type", "X-Requested-With", "Authorization", "Sentry-Trace", "Baggage", "x-elastic-client-meta", "x-swiftype-client", "x-swiftype-client-version", "Cookie"},
		AllowCredentials:   true,
		OptionsPassthrough: false,
		Debug:              true,
	}).Handler)
}

func SetLogs() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	logrus.SetLevel(level)
}

func getMaxRequestBodySize() int64 {
	_maxRequestSize := os.Getenv("MAX_REQUEST_BODY_SIZE")
	maxRequestSize, err := strconv.ParseInt(_maxRequestSize, 10, 64)
	if err != nil {
		maxRequestSize = 1024 * 1024 * 10 // 10MB
	}
	return maxRequestSize
}

func Serve(router *mux.Router) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("http://0.0.0.0:%s", port)

	err := http.ListenAndServe(fmt.Sprintf(":%s", port), http.MaxBytesHandler(router, getMaxRequestBodySize()))
	if err != nil {
		msg := fmt.Sprintf("calling ListenAndServe: %s", err)
		log.Fatal(msg)
	}
}
