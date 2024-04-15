begin ;
create or replace FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';
-- create table users, books, sessions, borrow_history
-- create table users
 CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email  text NOT NULL UNIQUE,
    username  text NOT NULL UNIQUE,
    password  text NOT NULL,
    type  text NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    constraint validate_user_type CHECK (type IN ('admin','librarian','borrower'))
);
create trigger update_users_updated_at before UPDATE on users for each row execute procedure update_updated_at_column();
insert into users(email, username, password, type) values
('admin@stu.ptit.edu.vn','admin','admin','admin');

-- create table books
 CREATE TABLE IF NOT EXISTS books (
    id SERIAL PRIMARY KEY,
    title  text NOT NULL,
    author  text DEFAULT '',
    type  text DEFAULT '',
    count INT NOT NULL,
    cover  text DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    constraint unique_book_per_author UNIQUE(title, author)
);
create trigger update_books_updated_at before UPDATE on books for each row execute procedure update_updated_at_column();

-- create table sessions
 CREATE TABLE IF NOT EXISTS sessions (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    session_id  text NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
create trigger update_sessions_updated_at before UPDATE on sessions for each row execute procedure update_updated_at_column();

-- create table borrow_history
 CREATE TABLE IF NOT EXISTS borrow_history (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book_id INT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    borrowed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    returned_at TIMESTAMPTZ NOT NULL DEFAULT '2024-01-01'::timestamp
);

commit ;