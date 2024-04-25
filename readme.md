# LIBRARY MANAGEMENT SYSTEM

## 1. Purpose

Web based system for librarians to manage physically borrowed books

## 2. Functions

- Admins can add/ update/ delete librarians, librarians can add/ update/ delete borrowers
- All users can see all books. Books can have multiple replicas
- Librarians can mark a book as borrowed by borrowers
- Each borrower can borrow max 3 books (configurable)
- User can't borrow 2 items of the same book at the same time

## 3. Technologies used

### Backend

- golang for core service
- postgres for structured data
- minio for blob storage
- docker for deployment

### Frontend

- HTML, CSS, Javascript

# 4. Usage

- `cp .env.sample .env`
- start by `make buildup`
- open http://localhost:8080/fe/login.html, login with username/ password `admin`/`admin` then start playing around
- stop by `make destroy`

# 5. Development notes

Improvements needed:

- Cache session_id in redis
- Handle auth more strictly
- Set up load balancer for app
