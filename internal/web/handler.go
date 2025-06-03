package lwt

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"sysafari.com/customs/cguard/internal/config"
	"sysafari.com/customs/cguard/pkg/utils"
)

const TimeLayout = "20060102150405"

// DownloadLwtExcel
// Download excel for LWT
// @Summary      Download excel for LWT
// @Description  get file by filename
// @Tags         lwt
// @Accept       json
// @Produce      json
// @Param        filename   path      string  true  "LWT filename"
// @Param 		 download   query 	  int false "Download file"
// @Success      200
// @Failure      400
// @Router       /lwt/{filename} [get]
func DownloadLwtExcel(c echo.Context) error {
	cfg := config.GetConfig()
	tmpDir := cfg.LWT.Tmp.Dir
	if !utils.IsDir(tmpDir) {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("The lwt root directory: %s is not exists.", tmpDir))
	}
	filename := c.Param("filename")
	if filename == "" {
		return c.String(http.StatusBadRequest, fmt.Sprintf("The filename must be provided,but was empty."))
	}

	filepath, err := getFilePath(tmpDir, filename)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("The filename:%s format not support.", filename))
	}

	if !utils.IsExists(filepath) {
		return c.String(http.StatusNotFound, fmt.Sprintf("The file:%s not found.", filename))
	}

	if "1" == c.QueryParam("download") {
		return c.Attachment(filepath, filename)
	}
	return c.File(filepath)
}

func getFilePath(rootDir string, filename string) (string, error) {
	fn := strings.Split(filename, ".")[0]
	paths := strings.Split(fn, "_")
	timestamp := paths[len(paths)-1]
	if timestamp == "" {
		return "", errors.New(fmt.Sprintf("The filename:%s cannot get timestamp.", filename))
	}
	ftime, err := time.Parse(TimeLayout, timestamp)
	if err != nil {
		return "", err
	}
	filepath := fmt.Sprintf("%s/%d/%d/%s", rootDir, ftime.Year(), ftime.Month(), filename)

	return filepath, nil
}
