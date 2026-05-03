package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getActorUsername(c *gin.Context) (string, bool) {
	v, ok := c.Get("username")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return "", false
	}

	actorUsername, ok := v.(string)
	if !ok || actorUsername == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid username"})
		return "", false
	}

	return actorUsername, true
}

func getFile(c *gin.Context, name string) ([]byte, string, error) {
	fileHeader, err := c.FormFile(name)
	switch {
	case errors.Is(err, http.ErrMissingFile):
		return nil, "", nil
	case err != nil:
		return nil, "", err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	extension := fileHeader.Header.Get("Content-Type")

	return data, extension, nil
}
