package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	// Parse the form data
	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse the form", err)
		return
	}

	// Get the image data from the form
	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain thumbnail data", err)
		return
	}

	// Get media subtype
	mediaType := fileHeaders.Header.Get("Content-Type")
	mediaSubtype := strings.Split(mediaType, "/")[1]

	// create a file with unique file path
	filename := videoIDString + "." + mediaSubtype
	dst, err := os.Create(filepath.Join(cfg.assetsRoot, filename))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create a file", err)
		return
	}
	defer dst.Close()

	// copy image data content to the new file
	_, err = io.Copy(dst, fileData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy a file", err)
		return
	}

	// Get the video metadata from the SQLite database
	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain video metadata", err)
		return
	} else if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to the user", err)
		return
	}

	// Update the video metadata with a new thumbnail URL
	filePath := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, filename)
	videoMetadata.ThumbnailURL = &filePath

	// Update the record in the db
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update the video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
