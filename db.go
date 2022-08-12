package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"log"
	"os"
	"github.com/jackc/pgx/v4/pgxpool"
)

type User struct {
	username string
	chatID sql.NullInt64
	name string
	unit string
	parent string
	baby string
	level string
	likes string
	dislikes string
	nogos string
	remarks string
}

func connectDB() (*pgxpool.Pool, error) {
	db, err := pgxpool.Connect(context.Background(), databaseUrl)
	if err != nil {
		log.Fatalln(err)
		return nil, err
	}
	log.Println("Connected to database")
	return db, nil
}

func initDB(db *pgxpool.Pool) {
	db.Exec(context.Background(), "create table if not exists users (id serial primary key, username varchar(255) unique not null, chatid int unique, parent varchar(255) not null, baby varchar(255) not null, name varchar(255) not null, unit varchar(255) not null, level varchar(255) not null, likes text default '', dislikes text default '', nogos text default '', remarks text default '')")
	db.Exec(context.Background(), "create table if not exists state (id serial primary key, started boolean not null default false)")
	db.Exec(context.Background(), "insert into state (id, started) values (1, false)")
}

func devResetDB(db *pgxpool.Pool) { //don't use this in production
	db.Exec(context.Background(), "drop table users")
	db.Exec(context.Background(), "drop table state")
	initDB(db)
	log.Println("Reset database")
}

func devSeedDB(db *pgxpool.Pool) { //don't use this in production
	db.Exec(context.Background(), "delete from users")
	db.Exec(context.Background(), "insert into users (username, chatid, parent, baby, name, unit, level, likes, dislikes, nogos, remarks) values ('Dsicol', '578886370', 'blur_sotong', 'Jingwei555', 'Darren', '#13-01A', 'Mega', '', '', '', '')")
	db.Exec(context.Background(), "insert into users (username, chatid, parent, baby, name, unit, level, likes, dislikes, nogos, remarks) values ('blur_sotong', '79822501', 'Jingwei555', 'Dsicol', 'Sitong', '#13-12D', 'Mega', '', '', '', '')")
	db.Exec(context.Background(), "insert into users (username, chatid, parent, baby, name, unit, level, likes, dislikes, nogos, remarks) values ('Jingwei555', '782161320', 'Dsicol', 'blur_sotong', 'Jing Wei', '#13-01B', 'Mega', '', '', '', '')")
	db.Exec(context.Background(), "insert into state (id, started) values (1, true)")
	log.Println("Seeded database")
}

func populateDB(db *pgxpool.Pool) {
	csvFile, err := os.Open("data.csv")
	if err != nil {
		log.Println(err)
	}
	defer csvFile.Close()

	reader := csv.NewReader(csvFile)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		log.Println(err)
	}

	for i, record := range records {
		if i == 0 {
			continue
		}
		db.Exec(context.Background(), "update users set parent = $1, baby = $2, name = $3, unit = $4, level = $5, likes = $6, dislikes = $7, nogos = $8, remarks = $9 where username = $10", record[1], record[2], record[3], record[4], record[5], record[6], record[7], record[8], record[9], record[0])
		db.Exec(context.Background(), "insert into users (username, parent, baby, name, unit, level, likes, dislikes, nogos, remarks) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)", record[0], record[1], record[2], record[3], record[4], record[5], record[6], record[7], record[8], record[9])
	}
	log.Println("Populated database")
}

func getUser(db *pgxpool.Pool, username string) (User, error) {
	var user User
	err := db.QueryRow(context.Background(), "select username, chatid, parent, baby, name, unit, level, likes, dislikes, nogos, remarks from users where username = $1", username).Scan(&user.username, &user.chatID, &user.parent, &user.baby, &user.name, &user.unit, &user.level, &user.likes, &user.dislikes, &user.nogos, &user.remarks)
	if err != nil {
		log.Println(err)
		return user, err
	}
	return user, nil
}

func getParent(db *pgxpool.Pool, username string) (User, error) {
	var parent string
	err := db.QueryRow(context.Background(), "select parent from users where username = $1", username).Scan(&parent)
	if err != nil {
		log.Fatalln(err)
		return User{}, err
	}
	return getUser(db, parent)
}

func getBaby(db *pgxpool.Pool, username string) (User, error) {
	var baby string
	err := db.QueryRow(context.Background(), "select baby from users where username = $1", username).Scan(&baby)
	if err != nil {
		log.Fatalln(err)
		return User{}, err
	}
	return getUser(db, baby)
}

func checkUser(db *pgxpool.Pool, username string) bool {
	var count int
	db.QueryRow(context.Background(), "select count(*) from users where username = $1", username).Scan(&count)
	return count != 0
}

func setChatID(db *pgxpool.Pool, username string, chatID int64) {
	db.Exec(context.Background(), "update users set chatid = $1 where username = $2", int(chatID), username)
}

func startGame(db *pgxpool.Pool) {
	db.Exec(context.Background(), "update state set started = true where id = 1")
	log.Println("Started game")
}

func stopGame(db *pgxpool.Pool) {
	db.Exec(context.Background(), "update state set started = false where id = 1")
	log.Println("Stopped game")
}

func isGameStarted(db *pgxpool.Pool) bool {
	var started bool
	db.QueryRow(context.Background(), "select started from state where id = 1").Scan(&started)
	return started
}

func getAllUsernames(db *pgxpool.Pool) []string {
	var usernames []string
	rows, err := db.Query(context.Background(), "select username from users")
	if err != nil {
		log.Println(err)
	}
	for rows.Next() {
		var username string
		err := rows.Scan(&username)
		if err != nil {
			log.Println(err)
		}
		usernames = append(usernames, username)
	}
	return usernames
}

func getChatIDs(db *pgxpool.Pool) []int64 {
	var chatIDs []int64
	rows, err := db.Query(context.Background(), "select chatid from users")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var chatID sql.NullInt64
		err := rows.Scan(&chatID)
		if err != nil {
			log.Println(err)
		}
		if chatID.Valid {
			chatIDs = append(chatIDs, int64(chatID.Int64))
		}
	}
	return chatIDs
}

func getUsersWithoutChatID(db *pgxpool.Pool) []string {
	var usernames []string
	rows, err := db.Query(context.Background(), "select username from users where chatid is null")
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var username string
		err := rows.Scan(&username)
		if err != nil {
			log.Println(err)
		}
		usernames = append(usernames, username)
	}
	return usernames
}