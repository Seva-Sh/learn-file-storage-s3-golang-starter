package main

import (
	"fmt"
	"io"
	"net/http"

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

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(int64(maxMemory))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse the form", err)
		return
	}

	fileData, fileHeaders, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain thumbnail data", err)
		return
	}

	mediaType := fileHeaders.Header.Get("Content-Type")
	imgData, err := io.ReadAll(fileData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't read image data", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain video metadata", err)
		return
	} else if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to the user", err)
		return
	}

	thumbnail := thumbnail{
		data:      imgData,
		mediaType: mediaType,
	}
	videoThumbnails[videoID] = thumbnail
	url := fmt.Sprintf("http://localhost:%v/api/thumbnails/%v", cfg.port, videoID)
	videoMetadata.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update the video metadata", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMetadata)
}
