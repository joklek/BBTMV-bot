package database

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const CREATE_DB = `
BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "users" (
	"telegram_id"	INTEGER NOT NULL UNIQUE,
	"enabled"	INTEGER NOT NULL DEFAULT 0,
	"price_from"	INTEGER NOT NULL DEFAULT 0,
	"price_to"	INTEGER NOT NULL DEFAULT 0,
	"rooms_from"	INTEGER NOT NULL DEFAULT 0,
	"rooms_to"	INTEGER NOT NULL DEFAULT 0,
	"year_from"	INTEGER NOT NULL DEFAULT 0,
	"min_floor" INTEGER NOT NULL DEFAULT 0,
	"show_with_fee" INTEGER NOT NULL DEFAULT 0,
	"filter_by_district" INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY("telegram_id")
);
CREATE TABLE IF NOT EXISTS "posts" (
	"id"	INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT UNIQUE,
	"link"	TEXT NOT NULL UNIQUE,
	"last_seen"	INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS "districts" (
	"id"	INTEGER NOT NULL UNIQUE,
	"name"	INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY(id)
);
CREATE TABLE IF NOT EXISTS "user_districts" (
	"user_id"	INTEGER NOT NULL,
	"district_id"	INTEGER NOT NULL,
	FOREIGN KEY(user_id) REFERENCES users(telegram_id),
	FOREIGN KEY(district_id) REFERENCES districts(id)
);
CREATE UNIQUE INDEX IF NOT EXISTS "index_posts_link" ON "posts" (
	"link"
);
COMMIT;
`

type Database struct {
	db *sql.DB
}

func Open(path string) (*Database, error) {
	_, fileErr := os.Stat(path)
	d, err := sql.Open("sqlite3", "file:"+path+"?_mutex=full")
	if os.IsNotExist(fileErr) {
		_, err := d.Exec(CREATE_DB)
		if err != nil {
			panic(err)
		}
	}
	// migrate old data
	_, err2 := d.Query("SELECT filter_by_district FROM users LIMIT 1")
	if err2 != nil {
		_, err := d.Exec(CREATE_DB)
		if err != nil {
			panic(err)
		}
		d.Exec("ALTER TABLE users ADD COLUMN filter_by_district INTEGER NOT NULL DEFAULT 0")
		var districts = [...]string{"Antakalnis", "Baltupiai", "Buivydiškės", "Bukčiai", "Fabijoniškės", "Filaretai", "Grigiškės", "Jeruzalė", "Justiniškės", "Karoliniškės", "Lazdynai",
			"Lazdynėliai", "Markučiai", "Naujamiestis", "Naujininkai", "Naujoji Vilnia", "Pašilaičiai", "Paupys", "Pavilnys", "Pilaitė", "Rasos", "Santariškės", "Saulėtekis",
			"Senamiestis", "Šeškinė", "Šiaurės miestelis", "Šnipiškės", "Užupis", "Verkiai", "Vilkpėdė", "Viršuliškės", "Visoriai", "Žemieji Paneriai", "Žirmūnai", "Žvėrynas"}
		for _, name := range districts {
			d.Exec("INSERT INTO districts(name) VALUES(?)", name)
		}
	}
	// d.Exec("DELETE FROM posts")
	return &Database{d}, err
}

type User struct {
	TelegramID       int64
	Enabled          bool
	PriceFrom        int
	PriceTo          int
	RoomsFrom        int
	RoomsTo          int
	YearFrom         int
	MinFloor         int
	ShowWithFees     bool
	FilterByDistrict bool
}

func (d *Database) GetInterestedTelegramIDs(price, rooms, year int, floor int, district string, isWithFee bool) []int64 {
	if district == "" || !d.DistrictExists(district) {
		return d.getRegularInterestedIds(price, rooms, year, floor, isWithFee)
	}
	telegram_IDs := make([]int64, 0)
	interestedQuery := "SELECT telegram_id FROM users LEFT JOIN user_districts ud ON ud.user_id = telegram_id LEFT JOIN districts d ON d.id = ud.district_id " +
		"WHERE enabled=1 AND filter_by_district=1 AND ? >= price_from AND ? <= price_to AND ? >= rooms_from AND ? <= rooms_to AND ? >= year_from AND min_floor <= ? AND d.name = ? "
	interestedInAllQuery := "SELECT telegram_id FROM users " +
		"WHERE enabled=1 AND filter_by_district=0 AND ? >= price_from AND ? <= price_to AND ? >= rooms_from AND ? <= rooms_to AND ? >= year_from AND min_floor <= ? "
	if isWithFee {
		interestedQuery += "AND show_with_fee = 1"
		interestedInAllQuery += "AND show_with_fee = 1"
	}
	rows, err := d.db.Query(interestedQuery, price, price, rooms, rooms, year, floor, district)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var telegramID int64
		if err = rows.Scan(&telegramID); err != nil {
			panic(err)
		}
		telegram_IDs = append(telegram_IDs, telegramID)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	rows, err = d.db.Query(interestedInAllQuery, price, price, rooms, rooms, year, floor)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var telegramID int64
		if err = rows.Scan(&telegramID); err != nil {
			panic(err)
		}
		telegram_IDs = append(telegram_IDs, telegramID)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	return telegram_IDs
}

func (d *Database) getRegularInterestedIds(price, rooms, year int, floor int, isWithFee bool) []int64 {
	telegram_IDs := make([]int64, 0)
	query := "SELECT telegram_id FROM users WHERE enabled=1 AND ? >= price_from AND ? <= price_to AND ? >= rooms_from AND ? <= rooms_to AND ? >= year_from AND min_floor <= ? "
	if isWithFee {
		query += "AND show_with_fee = 1"
	}
	rows, err := d.db.Query(query, price, price, rooms, rooms, year, floor)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var telegramID int64
		if err = rows.Scan(&telegramID); err != nil {
			panic(err)
		}
		telegram_IDs = append(telegram_IDs, telegramID)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	return telegram_IDs
}

func (d *Database) EnsureUserInDB(telegramID int64) {
	query := "INSERT OR IGNORE INTO users(telegram_id) VALUES(?)"
	_, err := d.db.Exec(query, telegramID)
	if err != nil {
		log.Fatalln(err)
	}
}

func (d *Database) InDatabase(link string) bool {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) AS count FROM posts WHERE link=? LIMIT 1", link).Scan(&count)
	if err != nil {
		log.Fatalln(err)
	}
	if count <= 0 {
		return false
	}
	query := "UPDATE posts SET last_seen=? WHERE link=?"
	_, err = d.db.Exec(query, time.Now().Unix(), link)
	if err != nil {
		panic(err)
	}
	return true
}

func (d *Database) AddPost(link string) int64 {
	query := "INSERT INTO posts(link, last_seen) VALUES(?, ?)"
	res, err := d.db.Exec(query, link, time.Now().Unix())
	if err != nil {
		panic(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		panic(err)
	}
	return id
}

// Delete posts older than 30 days
func (d *Database) DeleteOldPosts() {
	query := "DELETE FROM posts WHERE last_seen < ?"
	_, err := d.db.Exec(query, time.Now().AddDate(0, 0, -30).Unix())
	if err != nil {
		panic(err)
	}
}

func (d *Database) GetUser(telegramID int64) *User {
	var u User
	query := "SELECT * FROM users WHERE telegram_id=?"
	err := d.db.QueryRow(query, telegramID).Scan(&u.TelegramID, &u.Enabled, &u.PriceFrom, &u.PriceTo, &u.RoomsFrom, &u.RoomsTo, &u.YearFrom, &u.MinFloor, &u.ShowWithFees, &u.FilterByDistrict)
	if err != nil {
		panic(err)
	}
	return &u
}

func (d *Database) UpdateUser(user *User) {
	query := "UPDATE users SET enabled=1, price_from=?, price_to=?, rooms_from=?, rooms_to=?, year_from=?, min_floor=?, show_with_fee=? WHERE telegram_id=?"
	_, err := d.db.Exec(query, user.PriceFrom, user.PriceTo, user.RoomsFrom, user.RoomsTo, user.YearFrom, user.MinFloor, user.ShowWithFees, user.TelegramID)
	if err != nil {
		panic(err)
	}
}

func (d *Database) Enabled(telegramID int64) bool {
	var enabled int
	query := "SELECT enabled FROM users WHERE telegram_id=? LIMIT 1"
	err := d.db.QueryRow(query, telegramID).Scan(&enabled)
	if err != nil {
		panic(err)
	}
	return enabled == 1
}

func (d *Database) SetEnabled(telegramID int64, enabled bool) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	query := "UPDATE users SET enabled=? WHERE telegram_id=?"
	_, err := d.db.Exec(query, enabledVal, telegramID)
	if err != nil {
		panic(err)
	}
}

type DistrictForUser struct {
	Id      int64
	Name    string
	Enabled bool
}

func (d *Database) DistrictExists(name string) bool {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) AS count FROM districts WHERE name=? LIMIT 1", name).Scan(&count)
	if err != nil {
		log.Fatalln(err)
	}
	return count > 0
}

func (d *Database) GetAllDistrictsForUser(telegramId int64) []DistrictForUser {
	query := "SELECT d.id, d.name, CASE WHEN du.user_id IS NULL THEN 0 ELSE 1 END AS enabled FROM districts d LEFT JOIN user_districts du ON d.id = du.district_id WHERE du.user_id = ? OR du.user_id IS NULL ORDER BY d.name ASC"
	rows, err := d.db.Query(query, telegramId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var list []DistrictForUser
	for rows.Next() {
		var d DistrictForUser
		if err = rows.Scan(&d.Id, &d.Name, &d.Enabled); err != nil {
			panic(err)
		}
		list = append(list, d)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	return list
}

func (d *Database) GetAllEnabledDistrictsForUser(telegramId int64) []string {
	query := "SELECT d.name FROM districts d LEFT JOIN user_districts du ON d.id = du.district_id WHERE du.user_id = ? ORDER BY d.name ASC"
	rows, err := d.db.Query(query, telegramId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var d string
		if err = rows.Scan(&d); err != nil {
			panic(err)
		}
		list = append(list, d)
	}
	if err = rows.Err(); err != nil {
		panic(err)
	}
	return list
}

func (d *Database) ToggleDistrictForUser(id int, telegramID int64) bool {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) AS count FROM user_districts WHERE user_id=? AND district_id = ? LIMIT 1", telegramID, id).Scan(&count)
	if err != nil {
		log.Fatalln(err)
	}
	if count <= 0 {
		d.db.Exec("INSERT INTO user_districts(user_id, district_id) VALUES (?,?)", telegramID, id)
		return true
	} else {
		d.db.Exec("DELETE FROM user_districts WHERE user_id=? AND district_id = ?", telegramID, id)
		return false
	}
}

func (d *Database) ClearDistricts(telegramID int64) {
	d.db.Exec("DELETE FROM user_districts WHERE user_id=?", telegramID)
}

func (d *Database) ToggleFilteringDistricts(telegramID int64, enabled bool) {
	enabledVal := 0
	if enabled {
		enabledVal = 1
	}
	query := "UPDATE users SET filter_by_district=? WHERE telegram_id=?"
	_, err := d.db.Exec(query, enabledVal, telegramID)
	if err != nil {
		panic(err)
	}
}
