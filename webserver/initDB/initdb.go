package initdb

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
)

const (
	dbPath     = "zeroAPI.db"  // you can replace this with your desired database file path
	tableName  = "auth_tokens" // dont change !!!
	createStmt = `
		CREATE TABLE IF NOT EXISTS auth_tokens (
			token TEXT PRIMARY KEY
		);
	`
	checktokencount = "SELECT COUNT(*) FROM auth_tokens"
	checktoken      = "SELECT COUNT(*) FROM auth_tokens WHERE token=?"
	inserttoken     = "INSERT INTO auth_tokens (token) VALUES ('%s')"
	getAPI          = "SELECT token FROM auth_tokens"
	ANSIgreen       = "\033[32m"
	ANSIred         = "\033[31m"
	ANSIesc         = "\033[0m"
)

type ColoredString string

func (s ColoredString) Green() string {
	return fmt.Sprintf("%s%s%s", ANSIgreen, s, ANSIesc)
}

func (s ColoredString) Red() string {
	return fmt.Sprintf("%s%s%s", ANSIred, s, ANSIesc)
}

func generateAPIKey(db *sql.DB) (string, error) {
	// Generate 8 random bytes
	var result int
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}

	apiKey := hex.EncodeToString(randomBytes)

	apiKey += hex.EncodeToString(randomBytes)

	err = db.QueryRow("SELECT COUNT(*) FROM auth_tokens WHERE token=?", apiKey).Scan(&result)

	if err != nil {
		return "", err
	}

	if result > 0 {
		//you somehow managed to generate two identical 16 char hex key
		log.Println("identical API key detected")
		return "", errors.New("identical API")
	}

	_, err = db.Exec(fmt.Sprintf(inserttoken, apiKey))

	if err != nil {
		return "", err
	}

	return apiKey, nil
}

func printAPIlist(db *sql.DB) error {
	rows, err := db.Query(getAPI)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var apiKey string
		if err := rows.Scan(&apiKey); err != nil {
			log.Println("Error scanning API key:", err)
			continue
		}
		log.Println(ColoredString(fmt.Sprintf("API key:%s", apiKey)).Green())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func checkvalue(db *sql.DB) bool {
	var result int
	err := db.QueryRow(checktokencount).Scan(&result)
	if err != nil {
		log.Fatal(err)
	}
	return result > 0
}

func tableExists(db *sql.DB, tableName string) bool {
	var result int
	err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&result)
	if err != nil {
		log.Fatal(err)
	}
	return result > 0
}

func InitDB() error {

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer func() {
		_, err := db.Exec("VACUUM")
		if err != nil {
			log.Fatalln(err)
		}
		db.Close()
	}()

	// Check if the table exists
	if !tableExists(db, tableName) {
		log.Printf(ColoredString(fmt.Sprintf("%s does not exist, creating one", dbPath)).Red())
		_, err := db.Exec(createStmt)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Table '%s' created successfully.\n", tableName)
		key, err := generateAPIKey(db)

		if err != nil {
			if err.Error() == "identical API" {
				log.Println("reinitializing Database")
				InitDB()
			} else {
				log.Fatalln(err)
			}
		}

		log.Println(ColoredString(fmt.Sprintf("created API key:%s", key)).Green())
		return nil
	} else {
		if checkvalue(db) {
			log.Println("API token/s already present")
			printAPIlist(db)
			return nil
		}
		log.Println(ColoredString("No API key found in the table").Red())

		key, err := generateAPIKey(db)

		if err != nil {
			if err.Error() == "identical API" {
				log.Println("reinitializing Database")
				InitDB()
			} else {
				log.Fatalln(err)
			}
		}

		log.Println("created API key:", key)

		return nil
	}

}
