# chirpy

## Introduction
Welcome to the Chirpy Database. This is a practice program in which I worked with HTTP writers and requests. Chirpy is a mock social network similar to Twitter in which users may create and post tweets. You can see all the http requests I coded information for on the main.go file. The users and chirps are stored in tables using PostgreSQL. You can POST, GET, DELETE, and PUT different information in the databases. 

## Install libraries
To run the program, you will need to have go and postgres installed.

You can install go using the webi installer. Run this in your terminal:
```bash
curl -sS https://webi.sh/golang | sh
```

You can install PostgreSQL on macOS with brew:
```bash
brew install postgresql@15
```
Or on windows or linux:
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

Ensure the installation worked with
```bash
psql --version
```
## Config
The program looks for a .gatorconfig.json file in the home directory. Create that file now, and store this code in it:
```
{"db_url":"url","current_user_name":"username"}
```
The username will be overwritten when you run the register command.
The db_url will be the one for your postgres system, and will look like: postgres://username:@localhost:5432/gator.
If you have a password, it will go after the username: and before the @
Add the following flags to the end: ?sslmode=disable, making it look like: postgres://username:@localhost:5432/gator?sslmode=disable.

## Clone
To clone the repo, enter this into the terminal
```bash
git clone https://github.com/jacobhuneke/chirpy.git
cd chirpy
```

## Database Migrations
To create the necessary tables, install [goose](https://github.com/pressly/goose) and run the following command from the `sql/schema` directory:
```bash
goose postgres <your_db_url> up
```
