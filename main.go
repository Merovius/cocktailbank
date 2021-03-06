package main

import (
	"bufio"
	"database/sql"
	"fmt"
	cconf "github.com/koebi/cocktails/config"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type cocktail struct {
	name    string
	zutaten map[string]float64
}

type fest struct {
	datum     string
	cocktails map[string]float64
}

func newFest() fest {
	return fest{
		cocktails: make(map[string]float64),
	}
}

func getString(prompt string) (string, error) {
	fmt.Printf(prompt)

	s := bufio.NewScanner(os.Stdin)
	if !s.Scan() {
		return "", s.Err()
	}
	return strings.TrimSpace(s.Text()), nil
}

func getFloat(prompt string) (float64, error) {
	fmt.Printf(prompt)

	s := bufio.NewScanner(os.Stdin)
	if !s.Scan() {
		return 0, s.Err()
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(s.Text()), 64)
	return f, err
}

func getInt(prompt string) (int, error) {
	fmt.Printf(prompt)

	s := bufio.NewScanner(os.Stdin)
	if !s.Scan() {
		return 0, s.Err()
	}

	i, err := strconv.Atoi(strings.TrimSpace(s.Text()))
	return i, err
}

func getZutaten() map[string]float64 {
	anzahl, err := getInt("Anzahl der Zutaten: ")
	if err != nil {
		log.Fatal(err)
	}

	zutaten := make(map[string]float64)

	for i := 0; i < anzahl; i++ {
		name, err := getString("Name der Zutat " + strconv.Itoa(i) + ": ")
		if err != nil {
			log.Fatal(err)
		}
		menge, err := getFloat("Menge: ")
		if err != nil {
			log.Fatal(err)
		}

		zutaten[name] = menge
	}
	return zutaten
}

func createCocktail(db *sql.DB) {
	var c cocktail
	var err error
	c.name, err = getString("Name: ")
	if err != nil {
		log.Fatal(err)
	}
	c.zutaten = getZutaten()

	_, err = db.Exec("INSERT INTO cocktails (name) values ($1);", c.name)
	if err != nil {
		fmt.Println("Inserting into cocktails failed, getting:")
		log.Fatal(err)
	}

	rows, err := db.Query("SELECT id FROM cocktails WHERE name = $1;", c.name)
	if err != nil {
		log.Fatal("Selecting id failed, getting", err)
	}

	var id int
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			log.Fatal("Scanning rows failed, getting", err)
		}
	}
	rows.Close()

	stmt, err := db.Prepare("INSERT INTO zutaten (name, menge, cocktail) values($1, $2, $3)")
	if err != nil {
		log.Fatal("Preparing insertion into zutaten failed:", err)
	}

	st, err := db.Prepare("INSERT INTO inventar (name) values($1)")
	if err != nil {
		log.Fatal("Preparing insertion into inventar failed:", err)
	}

	for zutat, menge := range c.zutaten {
		_, err := stmt.Exec(zutat, menge, id)
		if err != nil {
			log.Fatal("Inserting into Cocktailzutaten failed:", err)
		}

		_, err = st.Exec(zutat)
		if err != nil {
			log.Fatal("Inserting into Inventar failed:", err)
		}
	}
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "fest.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE cocktails (id INTEGER NOT NULL PRIMARY KEY ASC AUTOINCREMENT, name TEXT);")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE zutaten (name TEXT, menge FLOAT, cocktail INT);")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE inventar (name TEXT UNIQUE ON CONFLICT IGNORE, vorhanden FLOAT DEFAULT 0.0, preis INTEGER DEFAULT 0);")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("CREATE TABLE feste (datum TEXT, cocktails INTEGER, preis INTEGER DEFAULT 0, menge INTEGER DEFAULT 0);")
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func cocktailZutaten(db *sql.DB, cocktail string) map[string]float64 {

	rows, err := db.Query("SELECT id FROM cocktails WHERE name = $1", cocktail)
	if err != nil {
		log.Fatal("selecting cocktail id failed:", err)
	}

	var id int
	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			log.Fatal("scanning rows failed:", err)
		}
	}
	rows.Close()

	rows, err = db.Query("SELECT name, menge FROM zutaten WHERE cocktail = $1", id)
	if err != nil {
		log.Fatal("selecting from zutaten failed:", err)
	}

	zutaten := make(map[string]float64)
	for rows.Next() {
		var name string
		var menge float64
		if err := rows.Scan(&name, &menge); err != nil {
			log.Fatal("Scanning rows failed:", err)
		}
		zutaten[name] = menge
	}
	rows.Close()

	return zutaten
}

func getCocktails(db *sql.DB) []cocktail {
	rows, err := db.Query("SELECT name FROM cocktails")
	if err != nil {
		log.Fatal("Selecting cocktail name failed:", err)
	}

	var cocktails []cocktail

	for rows.Next() {
		var c cocktail

		if err := rows.Scan(&c.name); err != nil {
			log.Fatal("Scanning Rows failed:", err)
		}

		c.zutaten = cocktailZutaten(db, c.name)
		cocktails = append(cocktails, c)
	}

	rows.Close()
	return cocktails
}

func addInventory(db *sql.DB) {
	name, err := getString("Name: ")
	if err != nil {
		log.Fatal(err)
	}
	avail, err := getFloat("Vorhanden [l]: ")
	if err != nil {
		log.Fatal(err)
	}
	price, err := getInt("Preis [ct]: ")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("INSERT INTO inventar (name, vorhanden, preis) VALUES ($1, $2, $3)", name, avail, price)
	if err != nil {
		log.Fatal("Adding Inventory entry failed:", err)
	}
}

func getInventory(db *sql.DB) map[string]float64 {
	inventar := make(map[string]float64)

	rows, err := db.Query("SELECT name, vorhanden FROM inventar")
	if err != nil {
		log.Fatal("selecting inventory failed", err)
	}

	for rows.Next() {
		var n string
		var v float64

		if err := rows.Scan(&n, &v); err != nil {
			log.Fatal("scanning rows failed:", err)
		}
		inventar[n] = v
	}
	rows.Close()

	return inventar
}

func getInventoryPriceList(db *sql.DB) map[string]float64 {
	prices := make(map[string]float64)

	rows, err := db.Query("SELECT name, preis FROM inventar")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var n string
		var p float64

		if err := rows.Scan(&n, &p); err != nil {
			log.Fatal(err)
		}
		prices[n] = p
	}
	rows.Close()

	return prices
}

func updateAvailability(db *sql.DB) {
	name, err := getString("Name: ")
	if err != nil {
		log.Fatal(err)
	}
	avail, err := getFloat("Vorhanden [l]: ")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("UPDATE inventar SET vorhanden = $2 WHERE name = $1", name, avail)
	if err != nil {
		log.Fatal(err)
	}
}

func updatePrice(db *sql.DB) {
	name, err := getString("Name: ")
	if err != nil {
		log.Fatal(err)
	}
	price, err := getInt("Preis [ct]: ")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec("UPDATE inventar SET price = $2 WHERE name = $1", name, price)
	if err != nil {
		log.Fatal(err)
	}
}

func getFest(db *sql.DB) fest {
	f := newFest()
	f.datum = cconf.Current

	rows, err := db.Query("SELECT cocktails, menge FROM feste WHERE datum = $1", f.datum)
	if err != nil {
		log.Fatal("selecting cocktails failed:", err)
	}

	for rows.Next() {
		var c string
		var m float64

		if err := rows.Scan(&c, &m); err != nil {
			log.Fatal("scanning rows failed:", err)
		}
		f.cocktails[c] = m
	}
	rows.Close()

	return f
}

func genShoppingList(db *sql.DB) (map[string]float64, map[string]float64) {
	cocktails := getCocktails(db)
	inventar := getInventory(db)
	f := getFest(db)

	liste := make(map[string]float64)

	for _, c := range cocktails {
		for z, m := range c.zutaten {
			liste[z] += f.cocktails[c.name] * m
		}
	}

	for z, m := range inventar {
		liste[z] -= m
		if liste[z] <= 0 {
			liste[z] = 0
		}
	}

	prices := getInventoryPriceList(db)
	preisliste := make(map[string]float64)

	for z, m := range liste {
		preisliste[z] = m * prices[z]
	}

	return liste, preisliste
}

func updateInventory(db *sql.DB) {
	items := []string{"Item hinzufügen [i]", "Menge ändern [m]", "Preis ändern [p]"}
	for i := range items {
		fmt.Println(i)
	}

	c, err := getString("Auswahl: ")
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case c == "i":
		addInventory(db)
	case c == "m":
		updateAvailability(db)
	case c == "p":
		updatePrice(db)
	}
}

func setFest(db *sql.DB) {
	fmt.Println("Wähle Cocktails aus. Zur Verfügung stehen: ")

	cocktails := getCocktails(db)
	for i, c := range cocktails {
		fmt.Println(i, c.name)
	}
	choice, err := getString("Bitte Auswahl mit ',' trennen.")
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := db.Prepare("INSERT INTO feste (datum, cocktails, preis, menge) values ($1, $2, $3, $4)")
	if err != nil {
		log.Fatal(err)
	}

	for _, c := range strings.Split(choice, ",") {
		id, err := strconv.Atoi(c)
		if err != nil {
			log.Fatal(err)
		}

		menge, err := getInt("Wie viele " + cocktails[id].name + " sollen angeboten werden? ")
		if err != nil {
			log.Fatal(err)
		}
		preis, err := getFloat("Wie viel soll ein " + cocktails[id].name + " kosten [ct]? ")
		if err != nil {
			log.Fatal(err)
		}

		stmt.Exec(cconf.Current, cocktails[id].name, preis, menge)
	}
}

func mainMenu(db *sql.DB) {
	items := []string{"Cocktail erstellen [c]", "Inventar updaten [i]", "Festcocktails festlegen [f]", "Einkaufsliste erstellen [e]"}
	for _, i := range items {
		fmt.Println(i)
	}

	c, err := getString("Aktion: ")
	if err != nil {
		log.Fatal(err)
	}

	switch {
	case c == "c":
		createCocktail(db)
	case c == "i":
		updateInventory(db)
	case c == "f":
		setFest(db)
	case c == "e":
		liste, preisliste := genShoppingList(db)
		fmt.Println(liste)
		fmt.Println(preisliste)
	}
}

func main() {
	var db *sql.DB

	_, err := os.Stat("fest.db")
	if os.IsNotExist(err) {
		db = initDB()
	} else if err != nil {
		log.Fatal("Stat returned:", err)
	} else {
		db, err = sql.Open("sqlite3", "fest.db")
		if err != nil {
			log.Fatal("Opening Database Failed:", err)
		}
	}

	mainMenu(db)

	//	createCocktail(db)
	//	addInventory(db)
	//	liste, preisliste := genShoppingList(db)
	//  fmt.Println(liste, preisliste)
}
