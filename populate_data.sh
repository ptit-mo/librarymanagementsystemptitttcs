#!/bin/bash

set -eoux pipefail

echo "Populating some data so when starting the application we have some data to work with :)"

# login to get cookie
cookie="session=$(curl -X POST -H "Content-Type: application/json" -d '{"username":"ad","password":"secret"}' http://localhost:8080/login | jq -r .session_id)"
echo $cookie

# add some books
unzip images.zip
trap 'rm -rf images' EXIT
pushd images
declare -i cnt=0
declare -a book_ids
for file in *
do
  path="$(curl -H "cookie: ${cookie}" -v -F "file=@$file" http://localhost:8080/admin/uploadimage | jq -r .path)"
  book_id=$(curl 'http://localhost:8080/librarian/book' -H "Cookie: ${cookie}" \
  --data-raw "{\"title\":\"${cnt}\",\"author\":\"${cnt}\",\"count\":1,\"type\":\"fiction\",\"cover\":\"${path}\"}" | jq -r .id)
  cnt=$((cnt+1))
  book_ids+=($book_id)
done
popd

# add some users
declare -a user_ids
for i in {1..100}
do
    user_id=$(curl 'http://localhost:8080/admin/user' -H "Cookie: ${cookie}" \
  --data-raw "{\"username\":\"borrower${i}\",\"password\":\"pwd\",\"type\":\"borrower\", \"email\":\"borrower${i}@localhost.com\"}" | jq -r .id)
  user_ids+=($user_id)
done
for i in {1..10}
do
  curl 'http://localhost:8080/admin/user' -H "Cookie: ${cookie}" \
  --data-raw "{\"username\":\"librarian${i}\",\"password\":\"pwd\",\"type\":\"librarian\", \"email\":\"librarian${i}@localhost.com\"}"
done

# some users borrow some books
for uid in "${user_ids[@]:0:10}"
do
    for bid in "${book_ids[@]:0:5}"
    do
      curl 'http://localhost:8080/librarian/bookborrow' -H "Cookie: ${cookie}" \
      --data-raw "{\"book_id\":${bid},\"user_id\":${uid}}"
    done
done
