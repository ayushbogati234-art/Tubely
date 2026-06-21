package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "io"
    "mime"
    "net/http"
    "os"

    "github.com/aws/aws-sdk-go-v2/service/s3"
    
    "github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
    "github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(
	w http.ResponseWriter,
	r *http.Request,
) {
	const maxUploadSize = 1 << 30

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxUploadSize,
	)

	videoIDString := r.PathValue("videoID")

	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Invalid video ID",
			err,
		)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(
			w,
			http.StatusUnauthorized,
			"Couldn't get token",
			err,
		)
		return
	}

	userID, err := auth.ValidateJWT(
		token,
		cfg.jwtSecret,
	)
	if err != nil {
		respondWithError(
			w,
			http.StatusUnauthorized,
			"Couldn't validate JWT",
			err,
		)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't get video",
			err,
		)
		return
	}

	if video.UserID != userID {
		respondWithError(
			w,
			http.StatusUnauthorized,
			"Not owner",
			nil,
		)
		return
	}

	err = r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Couldn't parse form",
			err,
		)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Couldn't get video",
			err,
		)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Invalid content type",
			err,
		)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(
			w,
			http.StatusBadRequest,
			"Video must be MP4",
			nil,
		)
		return
	}

	tempFile, err := os.CreateTemp(
		"",
		"tubely-upload.mp4",
	)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't create temp file",
			err,
		)
		return
	}

	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't save temp file",
			err,
		)
		return
	}

	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't seek temp file",
			err,
		)
		return
	}

	processedPath, err := processVideoForFastStart(
	tempFile.Name(),
)
if err != nil {
	respondWithError(
		w,
		http.StatusInternalServerError,
		"Couldn't process video",
		err,
	)
	return
}

defer os.Remove(processedPath)

processedFile, err := os.Open(processedPath)
if err != nil {
	respondWithError(
		w,
		http.StatusInternalServerError,
		"Couldn't open processed video",
		err,
	)
	return
}
defer processedFile.Close()

	aspectRatio, err := getVideoAspectRatio(tempFile.Name())
    if err != nil {
	respondWithError(
		w,
		http.StatusInternalServerError,
		"Couldn't determine aspect ratio",
		err,
	)
	return
}



	randomBytes := make([]byte, 32)

	_, err = rand.Read(randomBytes)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't generate key",
			err,
		)
		return
	}

	key := aspectRatio + "/" +
	hex.EncodeToString(randomBytes) +
	".mp4"

	_, err = cfg.s3Client.PutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket:      &cfg.s3Bucket,
			Key:         &key,
			Body:        processedFile,
			ContentType: &mediaType,
		},
	)
	if err != nil {
	fmt.Printf("S3 ERROR: %v\n", err)

	respondWithError(
		w,
		http.StatusInternalServerError,
		"Couldn't upload to S3",
		err,
	)
	return
   }

   fmt.Println("S3 upload successful")
   fmt.Println("Key:", key)

	videoURL := fmt.Sprintf(
    "%s/%s",
    cfg.s3CfDistribution,
    key,
    )

video.VideoURL = &videoURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(
			w,
			http.StatusInternalServerError,
			"Couldn't update database",
			err,
		)
		return
	}

	



}