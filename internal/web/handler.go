package lwt

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"sysafari.com/customs/cguard/internal/config"
	"sysafari.com/customs/cguard/internal/model"
	"sysafari.com/customs/cguard/internal/service"
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
		return c.String(http.StatusBadRequest, "The filename must be provided,but was empty.")
	}

	filepath, err := utils.GetFilePathByFilename(tmpDir, filename, TimeLayout)
	if err != nil {
		return c.String(http.StatusBadRequest, fmt.Sprintf("The filename:%s format not support.", filename))
	}

	if !utils.IsExists(filepath) {
		return c.String(http.StatusNotFound, fmt.Sprintf("The file:%s not found.", filename))
	}

	if c.QueryParam("download") == "1" {
		return c.Attachment(filepath, filename)
	}
	return c.File(filepath)
}

// AdjustLwtParams
// Adjust LWT parameters and regenerate LWT file.IMPORTANT: The customs's LWT file must be generated first.
// @Summary      Adjust LWT parameters
// @Description  Adjust LWT calculation parameters and regenerate LWT file.IMPORTANT: The customs's LWT file must be generated first.
// @Tags         lwt
// @Accept       json
// @Produce      json
// @Param        request body model.RequestForLwtAdjustment true "LWT adjustment request"
// @Success      200  {object} model.ResponseForLwtAdjustment
// @Failure      400  {object} model.ResponseForLwtAdjustment
// @Router       /lwt/adjust [post]
func AdjustLwtParamsAndGenerateNewLwt(c echo.Context) error {
	var req model.RequestForLwtAdjustment
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, model.ResponseForLwtAdjustment{
			CustomsId: req.CustomsId,
			Status:    "error",
			Error:     "Invalid request format",
		})
	}

	// Generate LWT file with adjusted parameters
	ser := service.NewLwtAdjustService(req)
	filename, hasDiff, err := ser.AdjustLwtParamsAndGenerateNewLwt()
	if err != nil {
		return c.JSON(http.StatusBadRequest, model.ResponseForLwtAdjustment{
			CustomsId: req.CustomsId,
			Status:    "error",
			Error:     err.Error(),
		})
	}

	return c.JSON(http.StatusOK, model.ResponseForLwtAdjustment{
		CustomsId:   req.CustomsId,
		Status:      "success",
		LwtFilename: filename,
		HasDiff:     hasDiff,
	})
}
