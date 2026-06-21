package main

import (
	"fmt"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
		"crypto/rand"
	"encoding/base64"
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

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't get thumbnail", err)
		return
	}
	defer file.Close()
	contentType := fileHeader.Header.Get("Content-Type")

	mediaType := fileHeader.Header.Get("Content-Type")
	mediaType, _, err = mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Thumbnail must be a JPEG or PNG image",
			nil,
		)
		return
	}
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not video owner", nil)
		return
	}

	var extension string

	switch mediaType {
	case "image/png":
		extension = "png"
	case "image/jpeg":
		extension = "jpg"
	default:
		respondWithError(w, http.StatusBadRequest, "Unsupported image type", nil)
		return
	}

	randomBytes := make([]byte, 32)

_, err = rand.Read(randomBytes)
if err != nil {
	respondWithError(w, http.StatusInternalServerError, "Couldn't generate filename", err)
	return
}

randomName := base64.RawURLEncoding.EncodeToString(randomBytes)
fileName := fmt.Sprintf("%s.%s", randomName, extension)


	filePath := filepath.Join(cfg.assetsRoot, fileName)

	outFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create file", err)
		return
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save file", err)
		return
	}

	thumbnailURL := fmt.Sprintf(
		"http://localhost:%s/assets/%s",
		cfg.port,
		fileName,
	)

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
