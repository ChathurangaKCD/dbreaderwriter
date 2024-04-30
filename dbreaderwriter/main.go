package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func main() {
	mode := os.Getenv("DB_TYPE")
	if mode == "" {
		fmt.Printf("DB_TYPE is not set")
		os.Exit(1)
	}
	if mode == "pg" {
		params := url.Values{}
		username := os.Getenv("PG_USER")
		pwd := os.Getenv("PG_PASSWORD")
		host := os.Getenv("PG_HOST")
		port := os.Getenv("PG_PORT")
		dbname := os.Getenv("PG_DBNAME")
		if username == "" {
			fmt.Printf("PG_USER is not set")
			os.Exit(1)
		}
		if pwd == "" {
			fmt.Printf("PG_PASSWORD is not set")
			os.Exit(1)
		}
		if host == "" {
			fmt.Printf("PG_HOST is not set")
			os.Exit(1)
		}
		if port == "" {
			fmt.Printf("PG_PORT is not set")
			os.Exit(1)
		}
		if dbname == "" {
			fmt.Printf("PG_DBNAME is not set")
			os.Exit(1)
		}
		params.Set("database", dbname)
		caCertPath := os.Getenv("PG_SSL_CA_CERT")
		if caCertPath != "" {
			params.Set("sslmode", "verify-ca")
			params.Set("sslrootcert", caCertPath)
			fmt.Printf("Using CA cert: %s\n", caCertPath)
		} else {
			params.Set("sslmode", "require")
		}

		conn := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(username, pwd),
			Host:     fmt.Sprintf("%s:%s", host, port),
			RawQuery: params.Encode(),
		}
		runPg(conn.String())
	}
	if mode == "redis" {
		username := os.Getenv("REDIS_USER")
		pwd := os.Getenv("REDIS_PASSWORD")
		host := os.Getenv("REDIS_HOST")
		port := os.Getenv("REDIS_PORT")
		runRedis(&redis.Options{
			Username: username,
			Addr:     fmt.Sprintf("%s:%s", host, port),
			Password: pwd,
			DB:       0,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			MaxRetries: 10,
			PoolSize:   3,
			OnConnect: func(ctx context.Context, c *redis.Conn) error {
				return c.ClientSetName(ctx, "testclient").Err()
			},
		})
	}

}

func runRedis(options *redis.Options) {
	fmt.Println("Connecting to Redis")
	rdb := redis.NewClient(options)
	for {
		value := fmt.Sprintf("%d", time.Now().Unix())
		fmt.Println("Setting key to: ", value)
		err := rdb.Set(ctx, "key", value, 0).Err()
		if err != nil {
			fmt.Println("Error setting key: ", err)
		}
		val, err := rdb.Get(ctx, "key").Result()
		if err != nil {
			fmt.Println("Error getting key: ", err)
		}
		fmt.Println("The value of key is:", val)
		fmt.Println("-------------------------")
		time.Sleep(10 * time.Second)
	}
}

func runPg(connStr string) {
	// Connect to the database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the table exists, and create it if it doesn't
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS table1 (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) NOT NULL
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Table exists/created")

	for {
		if err := db.Ping(); err != nil {
			fmt.Println("failed to ping: ", err)
		}
		writeEnabled, _ := strconv.ParseBool(os.Getenv("PG_WRITE"))
		if writeEnabled {
			value := fmt.Sprintf("%d", time.Now().Unix())
			// Insert a row into the table
			insertSQL := fmt.Sprintf("INSERT INTO table1 (name) VALUES ('%s')", value)
			_, err = db.Exec(insertSQL)
			if err != nil {
				fmt.Println("Error inserting row: ", err)
			}
			fmt.Println("Inserted row successfully")
		}
		// Query the table
		rows, err := db.Query("SELECT id, name FROM table1 ORDER BY id DESC LIMIT 1")
		if err != nil {
			fmt.Println("Error querying table: ", err)
		}
		func() {
			defer rows.Close()
			fmt.Println("Queried table successfully")
			for rows.Next() {
				var id int
				var name string
				err = rows.Scan(&id, &name)
				if err != nil {
					fmt.Println("Error scanning row: ", err)
				}
				fmt.Println("The value of id is:", id)
				fmt.Println("The value of name is:", name)
			}
		}()
		fmt.Println("-------------------------")
		time.Sleep(10 * time.Second)
	}
}
