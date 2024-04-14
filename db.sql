-- create table users, books, sessions, borrow_history
-- create table users
 CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) NOT NULL UNIQUE,
    username VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    type VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (type IN ("admin","librarian","borrower"))
);
CREATE TRIGGER [UsersUpdateLastTime]
    AFTER UPDATE
    ON users
    FOR EACH ROW
    WHEN NEW.updated_at < OLD.updated_at    --- this avoid infinite loop
BEGIN
    UPDATE users SET updated_at=CURRENT_TIMESTAMP WHERE id=OLD.id;
END;
insert into users(email, username, password, type) values
('admin@stu.ptit.edu.vn','ad','secret','admin');

-- create table books
 CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title VARCHAR(255) NOT NULL,
    author VARCHAR(255) DEFAULT '',
    type VARCHAR(255) DEFAULT '',
    count INTEGER NOT NULL,
    cover VARCHAR(255) DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(title, author)
);
CREATE TRIGGER [BooksUpdateLastTime]
    AFTER UPDATE
    ON books
    FOR EACH ROW
    WHEN NEW.updated_at < OLD.updated_at    --- this avoid infinite loop
BEGIN
    UPDATE books SET updated_at=CURRENT_TIMESTAMP WHERE id=OLD.id;
END;


-- create table sessions
 CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER UNIQUE NOT NULL,
    session_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
CREATE TRIGGER [SessionUpdateLastTime]
    AFTER UPDATE
    ON sessions
    FOR EACH ROW
    WHEN NEW.created_at < OLD.created_at    --- this avoid infinite loop
BEGIN
    UPDATE sessions SET created_at=CURRENT_TIMESTAMP WHERE id=OLD.id;
END;


-- create table borrow_history
 CREATE TABLE IF NOT EXISTS borrow_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    book_id INTEGER NOT NULL,
    borrowed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    returned_at TIMESTAMP NOT NULL DEFAULT '0000-00-00 00:00:00',
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (book_id) REFERENCES books(id)
);
