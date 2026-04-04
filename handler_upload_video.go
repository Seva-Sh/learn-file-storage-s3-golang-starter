package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// setting max upload limit to 1GB
	const maxUploadSize = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// extract video ID
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// get token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couln't find JWT", err)
		return
	}

	// get user id via authentication
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couln't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couln't obtain the video", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "The video does not belong to the user", nil)
		return
	}

	// get the video data
	videoData, videoHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain video data", err)
		return
	}

	defer videoData.Close()

	// validate video type
	videoType, _, err := mime.ParseMediaType(videoHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain content type", err)
		return
	} else if videoType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong content type", nil)
		return
	}
	videoSubtype := strings.Split(videoType, "/")[1]

	// create temp file, defer its removal and closure
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't create a file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// copy video data to the temporary file
	_, err = io.Copy(tempFile, videoData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't copy to the file", err)
		return
	}

	// reset temp file pointer to the beginning
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't return to the start of the file", err)
		return
	}

	// get video aspect ratio
	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't obtain video ratio", err)
		return
	}

	// process video for fast start
	processedFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't process video for fast start", err)
		return
	}
	defer os.Remove(processedFilePath)

	// open the processed file
	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't open processed file path", err)
		return
	}
	defer processedFile.Close()

	var directory string
	switch aspectRatio {
	case "16:9":
		directory = "landscape"
	case "9:16":
		directory = "portrait"
	default:
		directory = "other"
	}

	// create a file with unique file path
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't return to the start of the file", err)
		return
	}
	encodedStr := hex.EncodeToString((key))
	filename := directory + "/" + encodedStr + "." + videoSubtype

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &filename,
		Body:        processedFile,
		ContentType: &videoType,
	})
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't put object", err)
		return
	}

	// build the string and update the video url and video in the database
	videoStr := "https://" + cfg.s3Bucket + ".s3." + cfg.s3Region + ".amazonaws.com/" + filename
	video.VideoURL = &videoStr

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't updated the video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
