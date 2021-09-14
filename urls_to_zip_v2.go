package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const ManifestFileName = "./tiny_sample_archive.json"

type UrlFile struct {
	Url      string
	Filename string
}

func main() {

	urlFiles := downloadAndParse()

	// Create a client to fetch from hither and yon
	var netClient = &http.Client{
		Timeout: time.Second * 120,
	}

	imageStreamHandler := func(w http.ResponseWriter, req *http.Request) {

		// Create zip writer that writes directly to ResponseWriter
		zw := zip.NewWriter(w)

		// Iterate over manifest JSON array, downloading and zipping each image file
		for _, urlFile := range urlFiles {

			// Download one image file, compress and add to archive
			fetchAndCompress(netClient, zw, urlFile)
		}

		err := zw.Close()
		if err != nil {
			log.Fatal(err)
		}

	}

	http.HandleFunc("/images", imageStreamHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}

func downloadAndParse() []UrlFile {

	jsonBlob, err := os.ReadFile(ManifestFileName)
	if err != nil {
		fmt.Println("manifest read error:", err)
	}

	// Parse JSON manifest file into struct form
	var urlFiles = make([]UrlFile, 0)
	err = json.Unmarshal(jsonBlob, &urlFiles)
	if err != nil {
		fmt.Printf("manifest unmarshalling error: %s\n%+v\n", err, jsonBlob)
	}

	return urlFiles

}

func fetchAndCompress(netClient *http.Client, zw *zip.Writer, urlFile UrlFile) {

	// Download this file
	fmt.Printf("UrlFile %s\n", urlFile.Filename)
	response, err := netClient.Get(urlFile.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()
	response.Body.Close()

	// Create entry in zip archive
	f, err := zw.Create(urlFile.Filename)
	if err != nil {
		log.Fatal(err)
	}

	// Write and compress file to archive
	_, err = f.Write([]byte(body))
	if err != nil {
		log.Fatal(err)
	}
}
