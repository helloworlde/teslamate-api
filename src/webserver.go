// TeslaMate 车辆数据 REST API（PostgreSQL + MQTT 状态）。OpenAPI 由 swag 生成；浏览器访问 /swagger 或 /swagger/index.html 打开 Swagger UI。
//
// @title TeslaMate API
// @version 1.0
// @description 与 TeslaMate 采集数据对齐的 JSON API；看板对照见仓库 `执行计划.md`。
// @license.name MIT
// @host localhost:8080
// @BasePath /
// @schemes http
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	docs "github.com/tobiasehlert/teslamateapi/src/docs"
)

const (
	headerAPIVersion  = "API-Version"
	dbTimestampFormat = "2006-01-02T15:04:05Z" // format used in postgres for dates
)

var (
	// application readyz endpoint value for k8s
	isReady *atomic.Value

	// setting TeslaMateApi parameters
	apiVersion = "unspecified"

	// defining db var
	db *sql.DB

	// app-settings
	appUsersTimezone *time.Location
)

// main function
func main() {
	// setup of readiness endpoint code
	isReady = &atomic.Value{}
	isReady.Store(false)

	// setting log parameters
	log.SetFlags(log.Ldate | log.Lmicroseconds)

	// setting application to ReleaseMode if DEBUG_MODE is false
	if !getEnvAsBool("DEBUG_MODE", false) {
		// setting GIN_MODE to ReleaseMode
		gin.SetMode(gin.ReleaseMode)
		log.Printf("[info] TeslaMateApi running in release mode.")
	} else {
		// setting GIN_MODE to DebugMode
		gin.SetMode(gin.DebugMode)
		log.Printf("[info] TeslaMateApi running in debug mode.")
	}

	// getting app-settings from environment
	appUsersTimezone, _ = time.LoadLocation(getEnv("TZ", "Europe/Berlin"))
	if gin.IsDebugging() {
		log.Println("[debug] TeslaMateApi appUsersTimezone:", appUsersTimezone)
	}

	// init of API with connection to database
	initDBconnection()
	defer db.Close()

	// Connect to the MQTT broker
	statusCache, err := startMQTT()
	if getEnvAsBool("DISABLE_MQTT", false) {
		log.Printf("[info] TeslaMateApi MQTT connection not established.")
	} else {
		if err != nil {
			log.Fatalf("[error] TeslaMateApi MQTT connection failed: %s", err)
		}
	}

	if teslaApiHost := getEnv("TESLA_API_HOST", ""); teslaApiHost != "" {
		log.Printf("[info] TESLA_API_HOST is set: %s", teslaApiHost)
	}

	// kicking off Gin in value r
	r := gin.Default()

	docs.SwaggerInfo.BasePath = "/"

	// gzip：排除 /swagger，避免压缩 Swagger UI 静态资源导致页面空白或脚本异常
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/swagger"})))

	r.Use(func(c *gin.Context) {
		c.Header(headerAPIVersion, apiVersion)
		c.Next()
	})

	// set 404 not found page
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, RespPageNotFound{Code: "PAGE_NOT_FOUND", Message: "Page not found"})
	})

	// disable proxy feature of gin
	_ = r.SetTrustedProxies(nil)

	// Gin 的 /swagger/*any 不会匹配仅「/swagger」或「/swagger/」，需显式重定向到 index.html
	r.GET("/swagger", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
	r.GET("/swagger/", func(c *gin.Context) { c.Redirect(http.StatusFound, "/swagger/index.html") })
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/", httpRoot)

	// TeslaMateApi /api endpoints
	api := r.Group("/api")
	{
		api.GET("/", httpAPIRoot)

		// TeslaMateApi /api/v1 endpoints
		v1 := api.Group("/v1")
		{
			v1.GET("/", httpAPIv1Root)

			// v1 /api/v1/cars endpoints
			v1.GET("/cars", TeslaMateAPICarsV1)
			v1.GET("/cars/:CarID", TeslaMateAPICarsV1)

			// v1 /api/v1/cars/:CarID/battery-health endpoints
			v1.GET("/cars/:CarID/battery-health", TeslaMateAPICarsBatteryHealthV1)

			v1.GET("/cars/:CarID/states", TeslaMateAPICarsStatesV1)
			v1.GET("/cars/:CarID/positions", TeslaMateAPICarsPositionsV1)

			v1.GET("/database", TeslaMateAPIDatabaseV1)

			v1.GET("/cars/:CarID/metrics/charging-stats/extra", TeslaMateAPICarsMetricsChargingStatsExtraV1)
			v1.GET("/cars/:CarID/metrics/drive-stats/extra", TeslaMateAPICarsMetricsDriveStatsExtraV1)
			v1.GET("/cars/:CarID/metrics/charging-stats", TeslaMateAPICarsMetricsChargingStatsV1)
			v1.GET("/cars/:CarID/metrics/drive-stats", TeslaMateAPICarsMetricsDriveStatsV1)
			v1.GET("/cars/:CarID/metrics/efficiency", TeslaMateAPICarsMetricsEfficiencyV1)
			v1.GET("/cars/:CarID/metrics/mileage", TeslaMateAPICarsMetricsMileageV1)
			v1.GET("/cars/:CarID/metrics/locations", TeslaMateAPICarsMetricsLocationsV1)
			v1.GET("/cars/:CarID/metrics/timeline", TeslaMateAPICarsMetricsTimelineV1)
			v1.GET("/cars/:CarID/metrics/vampire-drain", TeslaMateAPICarsMetricsVampireDrainV1)
			v1.GET("/cars/:CarID/metrics/statistics", TeslaMateAPICarsMetricsStatisticsV1)
			v1.GET("/cars/:CarID/metrics/charge-level", TeslaMateAPICarsMetricsChargeLevelV1)
			v1.GET("/cars/:CarID/metrics/projected-range", TeslaMateAPICarsMetricsProjectedRangeV1)
			v1.GET("/cars/:CarID/metrics/overview", TeslaMateAPICarsMetricsOverviewV1)
			v1.GET("/cars/:CarID/metrics/states-analytics", TeslaMateAPICarsMetricsStatesAnalyticsV1)
			v1.GET("/cars/:CarID/metrics/visited", TeslaMateAPICarsMetricsVisitedV1)
			v1.GET("/cars/:CarID/metrics/dutch-tax", TeslaMateAPICarsMetricsDutchTaxV1)
			v1.GET("/cars/:CarID/metrics/trip", TeslaMateAPICarsMetricsTripV1)

			// v1 /api/v1/cars/:CarID/charges endpoints
			v1.GET("/cars/:CarID/charges", TeslaMateAPICarsChargesV1)
			v1.GET("/cars/:CarID/charges/current", TeslaMateAPICarsChargesCurrentV1)
			v1.GET("/cars/:CarID/charges/:ChargeID", TeslaMateAPICarsChargesDetailsV1)

			// v1 /api/v1/cars/:CarID/drives endpoints
			v1.GET("/cars/:CarID/drives", TeslaMateAPICarsDrivesV1)
			v1.GET("/cars/:CarID/drives/:DriveID", TeslaMateAPICarsDrivesDetailsV1)

			// v1 /api/v1/cars/:CarID/status endpoints
			v1.GET("/cars/:CarID/status", statusCache.TeslaMateAPICarsStatusV1)

			// v1 /api/v1/cars/:CarID/updates endpoints
			v1.GET("/cars/:CarID/updates", TeslaMateAPICarsUpdatesV1)

			// v1 /api/v1/globalsettings endpoints
			v1.GET("/globalsettings", TeslaMateAPIGlobalsettingsV1)
		}

		api.GET("/ping", apiPing)

		// health endpoints for kubernetes
		api.GET("/healthz", healthz)
		api.GET("/readyz", readyz)
	}

	// TeslaMateApi endpoints (before versioning)
	BasePathV1 := api.BasePath() + "/v1"
	r.GET("/cars", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/charges/:ChargeID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/drives/:DriveID", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/status", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/cars/:CarID/updates", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })
	r.GET("/globalsettings", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, BasePathV1+c.Request.RequestURI) })

	// build the http server
	server := &http.Server{
		Addr:    ":8080", // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
		Handler: r,
	}

	// setting readyz endpoint to true (if not using MQTT)
	if getEnvAsBool("DISABLE_MQTT", false) {
		isReady.Store(true)
	}

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	// we run a go routine that will receive the shutdown input
	go func() {
		<-quit
		log.Println("[info] TeslaMateAPI received shutdown input")
		if err := server.Close(); err != nil {
			log.Fatal("[error] TeslaMateAPI server close error:", err)
		}
	}()

	// run the server
	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("[info] TeslaMateAPI server gracefully shut down")
		} else {
			log.Fatal("[error] TeslaMateAPI server closed unexpectedly")
		}
	}
}

// initDBconnection func
func initDBconnection() {
	var err error

	// read environment variables with defaults for connection string
	dbhost := getEnv("DATABASE_HOST", "database")
	dbport := getEnvAsInt("DATABASE_PORT", 5432)
	dbuser := getEnv("DATABASE_USER", "teslamate")
	dbpass := getEnv("DATABASE_PASS", "secret")
	dbname := getEnv("DATABASE_NAME", "teslamate")
	dbtimeout := (getEnvAsInt("DATABASE_TIMEOUT", 60000) / 1000)
	dbsslmode := getEnv("DATABASE_SSL", "disable")
	dbsslrootcert := getEnv("DATABASE_SSL_CA_CERT_FILE", "")

	// convert boolean-like SSL mode for backwards compatibility
	switch dbsslmode {
	case "true", "noverify":
		dbsslmode = "require"
	case "false":
		dbsslmode = "disable"
	}

	// construct connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d", dbhost, dbport, dbuser, dbpass, dbname, dbsslmode, dbtimeout)

	// add SSL certificate configuration if provided
	if dbsslrootcert != "" {
		psqlInfo += " sslrootcert=" + dbsslrootcert
	}

	// open database connection
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("[error] initDBconnection - database connection error: %v", err)
	}

	// test database connection
	if err = db.Ping(); err != nil {
		log.Fatalf("[error] initDBconnection - database ping error: %v", err)
	}

	// showing database successfully connected
	if gin.IsDebugging() {
		log.Println("[debug] initDBconnection - database connection established successfully.")
	}
}

func TeslaMateAPIHandleErrorResponse(c *gin.Context, s1 string, s2 string, s3 string) {
	log.Println("[error] " + s1 + " - (" + c.Request.RequestURI + "). " + s2 + "; " + s3)
	c.JSON(http.StatusOK, RespAPIError{Error: s2})
}

func TeslaMateAPIHandleOtherResponse(c *gin.Context, httpCode int, s string, j interface{}) {
	// return successful response
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(httpCode, j)
}

func TeslaMateAPIHandleSuccessResponse(c *gin.Context, s string, j interface{}) {
	// print to log about request
	if gin.IsDebugging() {
		log.Println("[debug] " + s + " - (" + c.Request.RequestURI + ") returned data:")
		js, _ := json.Marshal(j)
		log.Printf("[debug] %s\n", js)
	}

	// return successful response
	log.Println("[info] " + s + " - (" + c.Request.RequestURI + ") executed successfully.")
	c.JSON(http.StatusOK, j)
}

func getTimeInTimeZone(datestring string) string {
	// parsing datestring into dbTimestampFormat
	t, _ := time.Parse(dbTimestampFormat, datestring)

	// formatting in users location in RFC3339 format
	ReturnDate := t.In(appUsersTimezone).Format(time.RFC3339)

	// logging time conversion to log
	if gin.IsDebugging() {
		log.Println("[debug] getTimeInTimeZone - UTC", t.Format(time.RFC3339), "time converted to", appUsersTimezone, "is", ReturnDate)
	}

	return ReturnDate
}

func parseDateParam(datestring string) (string, error) {
	if datestring == "" {
		return "", nil
	}

	// RFC3339 formats first — includes Z or timezone offset
	if t, err := time.Parse(time.RFC3339, datestring); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	// DateTime format (2006-01-02 15:04:05) without timezone info, interpret in user's timezone
	normalizedDateString := strings.ReplaceAll(datestring, "T", " ")
	if t, err := time.ParseInLocation(time.DateTime, normalizedDateString, appUsersTimezone); err == nil {
		return t.UTC().Format(dbTimestampFormat), nil
	}

	sanitizedInput := strings.NewReplacer("\n", "\\n", "\r", "\\r", "\t", "\\t").Replace(datestring)
	return "", fmt.Errorf("invalid date format: %s, please use RFC3339 format", sanitizedInput)
}

// getEnv func - read an environment or return a default value
func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultVal
}

// getEnvAsBool func - read an environment variable into a bool or return default value
func getEnvAsBool(name string, defaultVal bool) bool {
	valStr := getEnv(name, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

// getEnvAsInt func - read an environment variable into integer or return a default value
func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}

// convertStringToBool func - converts a string to boolean, returning false on failure
func convertStringToBool(data string) bool {
	value, err := strconv.ParseBool(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToBool: failed to parse '%s' as boolean - returning false", data)
		}
		return false
	}
	return value
}

// convertStringToFloat func - converts a string to float64, returning 0.0 on failure
func convertStringToFloat(data string) float64 {
	value, err := strconv.ParseFloat(data, 64)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToFloat: failed to parse '%s' as float64 - returning 0.0", data)
		}
		return 0.0
	}
	return value
}

// convertStringToInteger func - converts a string to int, returning 0 on failure
func convertStringToInteger(data string) int {
	value, err := strconv.Atoi(data)
	if err != nil {
		if gin.IsDebugging() {
			log.Printf("[warning] convertStringToInteger: failed to parse '%s' as integer - returning 0", data)
		}
		return 0
	}
	return value
}

// kilometersToMiles func
func kilometersToMiles(km float64) float64 {
	return (km * 0.62137119223733)
}

// kilometersToMilesNilSupport func
func kilometersToMilesNilSupport(km NullFloat64) NullFloat64 {
	km.Float64 = (km.Float64 * 0.62137119223733)
	return (km)
}

// milesToKilometers func
func milesToKilometers(mi float64) float64 {
	return (mi * 1.609344)
}

// kilometersToMilesInteger func
func kilometersToMilesInteger(km int) int {
	return int(float64(km) * 0.62137119223733)
}

// barToPsi func
func barToPsi(bar float64) float64 {
	return (bar * 14.503773800722)
}

// celsiusToFahrenheit func
func celsiusToFahrenheit(c float64) float64 {
	return (c*9/5 + 32)
}

// celsiusToFahrenheitNilSupport func
func celsiusToFahrenheitNilSupport(c NullFloat64) NullFloat64 {
	c.Float64 = (c.Float64*9/5 + 32)
	return (c)
}

// checkArrayContainsString func - check if string is inside stringarray
func checkArrayContainsString(s []string, e string) bool {
	return slices.Contains(s, e)
}

// healthz is a liveness probe.
// @Summary 存活探针
// @Tags system
// @Produce json
// @Success 200 {object} RespHealthz
// @Router /api/healthz [get]
func healthz(c *gin.Context) {
	c.JSON(http.StatusOK, RespHealthz{Status: http.StatusText(http.StatusOK)})
}

// readyz is a readiness probe.
// @Summary 就绪探针
// @Tags system
// @Produce json
// @Success 200 {object} RespReadyz
// @Failure 503 {object} RespAPIError
// @Router /api/readyz [get]
func readyz(c *gin.Context) {
	if isReady == nil || !isReady.Load().(bool) {
		c.JSON(http.StatusServiceUnavailable, RespAPIError{Error: http.StatusText(http.StatusServiceUnavailable)})
		return
	}
	TeslaMateAPIHandleSuccessResponse(c, "webserver", RespReadyz{Status: http.StatusText(http.StatusOK)})
}

// @Summary 根路径（服务运行提示）
// @Tags system
// @Produce json
// @Success 200 {object} RespHTTPRoot
// @Router / [get]
func httpRoot(c *gin.Context) {
	c.JSON(http.StatusOK, RespHTTPRoot{Message: "TeslaMateApi container running..", Path: "/"})
}

// @Summary /api 根路径
// @Tags system
// @Produce json
// @Success 200 {object} RespHTTPRoot
// @Router /api [get]
func httpAPIRoot(c *gin.Context) {
	c.JSON(http.StatusOK, RespHTTPRoot{Message: "TeslaMateApi container running..", Path: "/api"})
}

// @Summary API v1 根路径
// @Tags system
// @Produce json
// @Success 200 {object} RespHTTPRoot
// @Router /api/v1 [get]
func httpAPIv1Root(c *gin.Context) {
	c.JSON(http.StatusOK, RespHTTPRoot{Message: "TeslaMateApi v1 running..", Path: "/api/v1"})
}

// @Summary Ping
// @Tags system
// @Produce json
// @Success 200 {object} RespPing
// @Router /api/ping [get]
func apiPing(c *gin.Context) {
	c.JSON(http.StatusOK, RespPing{Message: "pong"})
}
