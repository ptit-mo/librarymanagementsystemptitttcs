#!/bin/sh

set -eoux pipefail

echo "Populating some data so when starting the application we have some data to work with :)"

# login to get cookie
cookie="session=$(curl -X POST -H "Content-Type: application/json" -d '{"username":"admin","password":"admin"}' http://localhost:8080/login | jq -r .session_id)"
echo $cookie

declare -i numborrower=1
declare -i numlibrarian=1

# add some books
unzip images.zip
trap 'rm -rf images' EXIT
declare -i cnt=0
declare -a book_ids
for file in images/*
do
  path="$(curl -H "cookie: ${cookie}" -v -F "file=@$file" http://localhost:8080/admin/uploadimage | jq -r .path)"
  if [ "$path" == "null" ]; then
    echo "Failed to upload image $file"
    exit 1
  fi
  book_id=$(curl 'http://localhost:8080/librarian/book' -H "Cookie: ${cookie}" --data-raw "{\"title\":\"book_${cnt}\",\"author\":\"author_${cnt}\",\"count\":1,\"type\":\"fiction\",\"cover\":\"${path}\"}" | jq -r .id)
  if [ "$book_id" == "null" ]; then
    echo "Failed to add book $cnt"
    exit 1
  fi
  cnt=$((cnt+1))
  book_ids+=($book_id)
done

# add some users
declare -a user_ids
borrower_password="brr"
for i in {1..10}
do
    user_id=$(curl 'http://localhost:8080/admin/user' -H "Cookie: ${cookie}" --data-raw "{\"username\":\"borrower${i}\",\"password\":\"${borrower_password}\",\"type\":\"borrower\", \"email\":\"borrower${i}@localhost.com\"}" | jq -r .id)
    if [ "$user_id" == "null" ]; then
      echo "Failed to add borrower $i"
      exit 1
    fi
  user_ids+=($user_id)
done
librarian_password="lbr"
for i in {1..3}
do
  curl 'http://localhost:8080/admin/user' -H "Cookie: ${cookie}" --data-raw "{\"username\":\"librarian${i}\",\"password\":\"${librarian_password}\",\"type\":\"librarian\", \"email\":\"librarian${i}@localhost.com\"}"
    if [ "$user_id" == "null" ]; then
      echo "Failed to add librarian $i"
      exit 1
    fi
done

# some users borrow some books
for uid in "${user_ids[@]:0:5}"
do
    for bid in "${book_ids[@]:0:3}"
    do
      curl 'http://localhost:8080/librarian/bookborrow' -H "Cookie: ${cookie}" --data-raw "{\"book_id\":${bid},\"user_id\":${uid}}"
      if [ $? -ne 0 ]; then
        echo "Failed to add borrow history $uid-$bid"
        exit 1
      fi
    done
done
