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

	// Create zip target file
	archive, err := os.CreateTemp("", ".zip")
	if err != nil {
		log.Fatal(err)
	}

	// Open second file descriptor for seeking and rewinding
	tmpArc, err := os.Open(archive.Name())
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(archive.Name()) // clean up

	// Make it storage for the new zip writer
	zw := zip.NewWriter(archive)

	// Create a client to fetch from hither and yon
	var netClient = &http.Client{
		Timeout: time.Second * 10,
	}

	var startPos, endPos = int64(0), int64(0)

	imageStreamHandler := func(w http.ResponseWriter, req *http.Request) {

		// Iterate over manifest JSON array, downloading and zipping each image file
		for _, urlFile := range urlFiles {

			startPos, _ = tmpArc.Seek(0, os.SEEK_CUR)
			fetchAndCompress(netClient, zw, urlFile)
			endPos, _ = tmpArc.Seek(0, os.SEEK_END)

			returnBuf := make([]byte, endPos-startPos)
			readSize, err := tmpArc.ReadAt(returnBuf, startPos)
			if err != nil {
				log.Fatal(err)
			}
			println("Readsize from tmparc: ", readSize)

			_, err = w.Write(returnBuf)
			if err != nil {
				log.Fatal(err)
			}

		}

		startPos, _ = tmpArc.Seek(0, os.SEEK_CUR)
		err = zw.Close()
		if err != nil {
			log.Fatal(err)
		}
		endPos, _ = tmpArc.Seek(0, os.SEEK_END)

		returnBuf := make([]byte, endPos-startPos)
		_, err := tmpArc.ReadAt(returnBuf, startPos)
		if err != nil {
			log.Fatal(err)
		}
		_, err = w.Write(returnBuf)
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
