package main

import (
	"database/sql"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsCommandV1 转发车辆指令至 Tesla API（需 ENABLE_COMMANDS 与 API_TOKEN）。
// @Summary 车辆指令 / 唤醒
// @Tags commands
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param Command path string false "指令名（POST /command/:Command）"
// @Success 200 {object} RespEnabledCommandNames "GET /command：可用指令列表"
// @Success 200 {object} TeslaUpstreamJSON "POST /command/*、/wake_up：Tesla API 原始 JSON"
// @Failure 401 {object} RespAPIError
// @Failure 403 {object} RespAPIError
// @Router /api/v1/cars/{CarID}/command [get]
// @Router /api/v1/cars/{CarID}/command/{Command} [post]
// @Router /api/v1/cars/{CarID}/wake_up [post]
// TeslaMateAPICarsCommandV1 func
func TeslaMateAPICarsCommandV1(c *gin.Context) {

	// creating required vars
	var (
		CarsCommandsError1                                 = "Unable to load cars."
		TeslaAccessToken, TeslaVehicleID, TeslaEndpointUrl string
		err                                                error
	)

	// check if commands are enabled.. if not we need to abort
	if !getEnvAsBool("ENABLE_COMMANDS", false) {
		log.Println("[warning] TeslaMateAPICarsCommandV1 ENABLE_COMMANDS is not true.. returning 403 forbidden.")
		TeslaMateAPIHandleOtherResponse(c, http.StatusForbidden, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "You are not allowed to access commands"})
		return
	}

	// if request method is GET return list of commands
	if c.Request.Method == http.MethodGet {
		TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsCommandV1", RespEnabledCommandNames{EnabledCommands: allowList})
		return
	}

	// authentication for the endpoint
	validToken, errorMessage := validateAuthToken(c)
	if !validToken {
		TeslaMateAPIHandleOtherResponse(c, http.StatusUnauthorized, "TeslaMateAPICarsCommandV1", RespAPIError{Error: errorMessage})
		return
	}

	// getting CarID param from URL and validating that it's not zero
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		log.Println("[error] TeslaMateAPICarsCommandV1 CarID is invalid (zero)!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusBadRequest, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "CarID invalid"})
		return
	}

	// getting request body to pass to Tesla
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in first io.ReadAll", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "internal io reading error"})
		return
	}

	// getting :Command
	command := ("/command/" + c.Param("Command"))
	// if command is /command/ or /command/wake_up, set to /wake_up only
	if command == "/command/" || command == "/command/wake_up" {
		command = "/wake_up"
	}

	if !checkArrayContainsString(allowList, command) {
		log.Println("[warning] TeslaMateAPICarsCommandV1 command not allowed!")
		TeslaMateAPIHandleOtherResponse(c, http.StatusUnauthorized, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "unauthorized"})
		return
	}

	// get TeslaVehicleID and TeslaAccessToken
	query := `
		SELECT
			eid as TeslaVehicleID,
			(SELECT access FROM private.tokens LIMIT 1) as TeslaAccessToken
		FROM cars
		WHERE id = $1
		LIMIT 1;`
	row := db.QueryRow(query, CarID)

	err = row.Scan(
		&TeslaVehicleID,
		&TeslaAccessToken,
	)

	switch err {
	case sql.ErrNoRows:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsCommandV1", "No rows were returned!", err.Error())
		return
	case nil:
		// nothing wrong.. continuing
		break
	default:
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsCommandV1", CarsCommandsError1, err.Error())
		return
	}

	// load ENCRYPTION_KEY environment variable
	teslaMateEncryptionKey := getEnv("ENCRYPTION_KEY", "")
	if teslaMateEncryptionKey == "" {
		log.Println("[error] TeslaMateAPICarsCommandV1 can't get ENCRYPTION_KEY.. will fail to perform command.")
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "missing ENCRYPTION_KEY env variable"})
		return
	}

	// decrypt access token
	TeslaAccessToken = decryptAccessToken(TeslaAccessToken, teslaMateEncryptionKey)

	switch getCarRegionAPI(TeslaAccessToken) {
	case ChinaAPI:
		TeslaEndpointUrl = getEnv("TESLA_API_HOST", "https://owner-api.vn.cloud.tesla.cn")
	default:
		TeslaEndpointUrl = getEnv("TESLA_API_HOST", "https://owner-api.teslamotors.com")
	}

	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPost, TeslaEndpointUrl+"/api/1/vehicles/"+TeslaVehicleID+command, strings.NewReader(string(reqBody)))
	req.Header.Set("Authorization", "Bearer "+TeslaAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TeslaMateApi/"+apiVersion+" (+https://github.com/tobiasehlert/teslamateapi)")
	resp, err := client.Do(req)

	// check response error
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in http request to "+TeslaEndpointUrl, err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "internal http request error"})
		return
	}

	defer resp.Body.Close()
	defer client.CloseIdleConnections()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[error] TeslaMateAPICarsCommandV1 error in second io.ReadAll:", err)
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsCommandV1", RespAPIError{Error: "internal io reading error"})
		return
	}
	TeslaMateAPIHandleOtherResponse(c, resp.StatusCode, "TeslaMateAPICarsCommandV1", TeslaUpstreamJSON(respBody))

}
