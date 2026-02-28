package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Photo struct {
	AlbumID      int    `json:"albumId"`
	ID           int    `json:"id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

// ANSI color constants
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Magenta = "\033[35m"
	Red    = "\033[31m"
)

func main() {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get("https://jsonplaceholder.typicode.com/photos")
	if err != nil {
		fmt.Println(Red, "Error haciendo request:", err, Reset)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(Red, "HTTP error:", resp.Status, Reset)
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(Red, "Error leyendo body:", err, Reset)
		os.Exit(1)
	}

	var photos []Photo
	if err := json.Unmarshal(body, &photos); err != nil {
		fmt.Println(Red, "Error parseando JSON:", err, Reset)
		os.Exit(1)
	}

	printHeader(len(photos))

	for _, p := range photos {
		printPhoto(p)
	}
}

func printHeader(total int) {
	fmt.Println(Bold + Cyan + "============================================")
	fmt.Println("        JSONPlaceholder Photos Client       ")
	fmt.Println("============================================" + Reset)
	fmt.Printf(Bold+"Total registros: "+Green+"%d"+Reset+"\n\n", total)
}

func printPhoto(p Photo) {
	fmt.Println(Bold + Blue + "--------------------------------------------" + Reset)
	fmt.Printf(Bold+Yellow+"Album ID: "+Reset+"%d\n", p.AlbumID)
	fmt.Printf(Bold+Yellow+"ID: "+Reset+"%d\n", p.ID)
	fmt.Printf(Bold+Yellow+"Title: "+Reset+"%s\n", p.Title)
	fmt.Printf(Bold+Yellow+"URL: "+Green+"%s"+Reset+"\n", p.URL)
	fmt.Printf(Bold+Yellow+"Thumbnail: "+Magenta+"%s"+Reset+"\n", p.ThumbnailURL)
}