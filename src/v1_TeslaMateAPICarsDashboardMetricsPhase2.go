package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsMetricsStatisticsV1 对齐 Statistics 看板「per period」多表（驾驶/充电/能耗）。
// @Summary 周期统计（Statistics）
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param period query string true "day|week|month|year"
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Param preferredRange query string false "ideal|rated"
// @Param timezone query string false "IANA 时区，默认 TZ 环境变量"
// @Param highPrecision query int false "refId D：0=事件流 1=positions 细粒度毛电耗" Enums(0,1) default(0)
// @Success 200 {object} RespMetricsStatistics
// @Router /api/v1/cars/{CarID}/metrics/statistics [get]
func TeslaMateAPICarsMetricsStatisticsV1(c *gin.Context) {
	const errMsg = "Unable to load statistics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, "invalid CarID")
		return
	}
	period, err := validateStatisticsPeriod(c.Query("period"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", err.Error(), err.Error())
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", "Invalid date format.", err.Error())
		return
	}
	defPR, _, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	if _, err := validatePreferredRangeColumn(pr); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", err.Error(), err.Error())
		return
	}
	tz := safeTimezoneForSQL(c)
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)

	sqlA := fmt.Sprintf(`
WITH data AS (
SELECT
  d.duration_min > 1 AND d.distance > 1 AND
  (start_position.usable_battery_level IS NULL OR (end_position.battery_level - end_position.usable_battery_level) = 0) AS is_sufficiently_precise,
  d.%[1]s - d.%[2]s AS range_diff,
  date_trunc('%[3]s', timezone('UTC', d.start_date), '%[4]s') AS date,
  d.*
FROM drives d
LEFT JOIN positions start_position ON d.start_position_id = start_position.id
LEFT JOIN positions end_position ON d.end_position_id = end_position.id
)
SELECT
  EXTRACT(EPOCH FROM date)*1000 AS date_from,
  EXTRACT(EPOCH FROM date + interval '1 %[3]s')*1000 AS date_to,
  CASE '%[3]s'
    WHEN 'month' THEN to_char(timezone('%[4]s', date), 'YYYY Month')
    WHEN 'year' THEN to_char(timezone('%[4]s', date), 'YYYY')
    WHEN 'week' THEN 'week ' || to_char(timezone('%[4]s', date), 'WW') || ' starting ' || to_char(timezone('%[4]s', date), 'YYYY-MM-DD')
    ELSE to_char(timezone('%[4]s', date), 'YYYY-MM-DD')
  END AS display,
  date,
  sum(duration_min)*60 AS sum_duration_h,
  convert_km(max(end_km)::numeric - min(start_km)::numeric, (SELECT unit_of_length FROM settings LIMIT 1)) AS sum_distance,
  convert_celsius(avg(outside_temp_avg), (SELECT unit_of_temperature FROM settings LIMIT 1)) AS avg_outside_temp,
  count(*) AS cnt,
  CASE WHEN sum(range_diff) > 0 THEN sum(distance)/sum(range_diff) ELSE NULL END AS efficiency
FROM data
WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
GROUP BY date`, colS, colE, period, tz)

	sqlB := fmt.Sprintf(`
WITH data AS (
  SELECT charging_processes.*,
    date_trunc('%[1]s', timezone('UTC', start_date), '%[2]s') AS date
  FROM charging_processes
)
SELECT
  EXTRACT(EPOCH FROM date)*1000 AS date_from,
  EXTRACT(EPOCH FROM date + interval '1 %[1]s')*1000 AS date_to,
  CASE '%[1]s'
    WHEN 'month' THEN to_char(timezone('%[2]s', date), 'YYYY Month')
    WHEN 'year' THEN to_char(timezone('%[2]s', date), 'YYYY')
    WHEN 'week' THEN 'week ' || to_char(timezone('%[2]s', date), 'WW') || ' starting ' || to_char(timezone('%[2]s', date), 'YYYY-MM-DD')
    ELSE to_char(timezone('%[2]s', date), 'YYYY-MM-DD')
  END AS display,
  date,
  sum(greatest(charge_energy_added, charge_energy_used)) AS sum_energy_used_kwh,
  sum(charge_energy_added) AS sum_energy_added_kwh,
  sum(greatest(charge_energy_added, charge_energy_used)) / count(*) AS avg_energy_charged_kwh,
  sum(cost) AS cost_charges,
  count(*) AS cnt_charges
FROM data
WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  AND (charge_energy_added IS NULL OR charge_energy_added > 0.1)
GROUP BY date`, period, tz)

	sqlC := fmt.Sprintf(`
WITH data AS (
  SELECT drives.*,
    date_trunc('%[1]s', timezone('UTC', start_date), '%[2]s') AS date
  FROM drives
)
SELECT
  EXTRACT(EPOCH FROM date)*1000 AS date_from,
  EXTRACT(EPOCH FROM date + interval '1 %[1]s')*1000 AS date_to,
  CASE '%[1]s'
    WHEN 'month' THEN to_char(timezone('%[2]s', date), 'YYYY Month')
    WHEN 'year' THEN to_char(timezone('%[2]s', date), 'YYYY')
    WHEN 'week' THEN 'week ' || to_char(timezone('%[2]s', date), 'WW') || ' starting ' || to_char(timezone('%[2]s', date), 'YYYY-MM-DD')
    ELSE to_char(timezone('%[2]s', date), 'YYYY-MM-DD')
  END AS display,
  date,
  sum((data.%[3]s - data.%[4]s) * car.efficiency * 1000) /
    convert_km(sum(distance)::numeric, (SELECT unit_of_length FROM settings LIMIT 1)) AS consumption_net
FROM data
JOIN cars car ON car.id = car_id
WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
GROUP BY date`, period, tz, colS, colE)

	rowsA, err := db.Query(sqlA, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}
	defer rowsA.Close()
	drivesPeriod, err := scanStatisticsDrivePeriod(rowsA)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}

	rowsB, err := db.Query(sqlB, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}
	defer rowsB.Close()
	chargesPeriod, err := scanStatisticsChargePeriod(rowsB)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}

	rowsC, err := db.Query(sqlC, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}
	defer rowsC.Close()
	consumptionPeriod, err := scanStatisticsConsumptionNet(rowsC)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}

	highPrec := 0
	if strings.TrimSpace(c.Query("highPrecision")) == "1" {
		highPrec = 1
	}
	sqlD := statisticsRefIDGrossQuery(pr, period, tz, highPrec)
	rowsD, err := db.Query(sqlD, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}
	defer rowsD.Close()
	consumptionGross, err := scanStatisticsConsumptionGross(rowsD)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsStatisticsV1", RespMetricsStatistics{
		Data: APIMetricsStatisticsData{
			DrivesPerPeriod:        drivesPeriod,
			ChargesPerPeriod:       chargesPeriod,
			ConsumptionNetPeriod:   consumptionPeriod,
			ConsumptionGrossPeriod: consumptionGross,
			Period:                 period,
			Timezone:               tz,
			PreferredRange:         pr,
			HighPrecision:          highPrec,
			WindowStartUtc:         start,
			WindowEndUtc:           end,
		},
	})
}

// TeslaMateAPICarsMetricsChargeLevelV1 对齐 Charge Level 看板（positions 分桶平均 SOC）。
// @Summary 充电/停车电量曲线（分桶）
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param bucketMinutes query int false "分桶分钟数" default(2)
// @Param startDate query string false "开始（同时作 date_bin 对齐原点，同看板 $__from）"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsChargeLevel
// @Router /api/v1/cars/{CarID}/metrics/charge-level [get]
func TeslaMateAPICarsMetricsChargeLevelV1(c *gin.Context) {
	const errMsg = "Unable to load charge level series."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargeLevelV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargeLevelV1", "Invalid date format.", err.Error())
		return
	}
	bm := convertStringToInteger(c.DefaultQuery("bucketMinutes", "2"))
	if bm < 1 {
		bm = 2
	}
	if bm > 60 {
		bm = 60
	}

	// 与 dashboards/charge-level.json 主序列一致：date_bin(..., timezone('UTC', date), to_timestamp(__from))；API 用 $4=查询起点。
	q := fmt.Sprintf(`
SELECT
  date_bin('%d minutes'::interval, timezone('UTC', date), $4::timestamptz) AS bucket_time,
  avg(battery_level) AS battery_level,
  avg(usable_battery_level) AS usable_battery_level
FROM positions
WHERE car_id = $1 AND date >= $2 AND date <= $3 AND ideal_battery_range_km IS NOT NULL
GROUP BY 1
ORDER BY 1`, bm)
	rows, err := db.Query(q, CarID, start, end, start)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargeLevelV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	series, err := scanChargeLevelSeries(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargeLevelV1", errMsg, err.Error())
		return
	}

	thQ := `SELECT 20 AS lower, CASE WHEN lfp_battery THEN 100 ELSE 80 END AS upper
		FROM cars INNER JOIN car_settings ON cars.settings_id = car_settings.id WHERE cars.id = $1`
	var lower, upper sql.NullInt64
	_ = db.QueryRow(thQ, CarID).Scan(&lower, &upper)

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsChargeLevelV1", RespMetricsChargeLevel{
		Data: APIMetricsChargeLevelData{
			Series:         series,
			BucketMinutes:  bm,
			Thresholds:     APIMetricsChargeLevelThresholds{Lower: sqlNullInt64Ptr(lower), Upper: sqlNullInt64Ptr(upper)},
			WindowStartUtc: start,
			WindowEndUtc:   end,
		},
	})
}

// TeslaMateAPICarsMetricsProjectedRangeV1 对齐 Projected Range 看板多条序列。
// @Summary 表显续航/里程/外温聚合
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param interval query string false "如 5 minutes, 1 hour, 1 day" default(1 hour)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Param preferredRange query string false "ideal|rated"
// @Success 200 {object} RespMetricsProjectedRange
// @Router /api/v1/cars/{CarID}/metrics/projected-range [get]
func TeslaMateAPICarsMetricsProjectedRangeV1(c *gin.Context) {
	const errMsg = "Unable to load projected range."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", "Invalid date format.", err.Error())
		return
	}
	iv, err := validateProjectedInterval(c.DefaultQuery("interval", "1 hour"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", err.Error(), err.Error())
		return
	}
	defPR, _, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	brCol := pr + "_battery_range_km"

	qMileage := fmt.Sprintf(`
SELECT
  date_bin('%[1]s'::interval, timezone('UTC', date), timestamptz '2000-01-01') AS time,
  convert_km(avg(odometer)::numeric, (SELECT unit_of_length FROM settings LIMIT 1)) AS mileage
FROM positions
WHERE date >= $2 AND date <= $3 AND car_id = $1 AND ideal_battery_range_km IS NOT NULL
GROUP BY 1 ORDER BY 1`, iv)

	qBatt := fmt.Sprintf(`
SELECT
  date_bin('%[1]s'::interval, timezone('UTC', date), timestamptz '2000-01-01') AS time,
  avg(battery_level) AS battery_level,
  avg(coalesce(usable_battery_level, battery_level)) AS usable_battery_level
FROM (
  SELECT battery_level, usable_battery_level, date FROM positions
  WHERE car_id = $1 AND date >= $2 AND date <= $3 AND ideal_battery_range_km IS NOT NULL
  UNION ALL
  SELECT battery_level, null::smallint, c.date FROM charges c
  JOIN charging_processes p ON p.id = c.charging_process_id
  WHERE p.car_id = $1 AND c.date >= $2 AND c.date <= $3
) AS data
GROUP BY 1 ORDER BY 1`, iv)

	qTemp := fmt.Sprintf(`
SELECT
  date_bin('%[1]s'::interval, timezone('UTC', date), timestamptz '2000-01-01') AS time,
  avg(convert_celsius(outside_temp, (SELECT unit_of_temperature FROM settings LIMIT 1))) AS outside_temp
FROM positions
WHERE date >= $2 AND date <= $3 AND car_id = $1 AND ideal_battery_range_km IS NOT NULL
GROUP BY 1 ORDER BY 1`, iv)

	qProj := fmt.Sprintf(`
SELECT
  date_bin('%[2]s'::interval, timezone('UTC', date), timestamptz '2000-01-01') AS time,
  convert_km((sum(data.%[1]s) / nullif(sum(coalesce(data.usable_battery_level, data.battery_level)), 0) * 100)::numeric,
    (SELECT unit_of_length FROM settings LIMIT 1)) AS projected_range_per_soc
FROM (
  SELECT battery_level, usable_battery_level, date, rated_battery_range_km, ideal_battery_range_km
  FROM positions WHERE car_id = $1 AND date >= $2 AND date <= $3 AND ideal_battery_range_km IS NOT NULL
  UNION ALL
  SELECT battery_level, coalesce(usable_battery_level, battery_level), c.date, rated_battery_range_km, ideal_battery_range_km
  FROM charges c JOIN charging_processes p ON p.id = c.charging_process_id
  WHERE c.date >= $2 AND c.date <= $3 AND p.car_id = $1
) AS data
GROUP BY 1
ORDER BY 1`, brCol, iv)

	rM, err := db.Query(qMileage, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}
	defer rM.Close()
	mileage, err := scanProjMileage(rM)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}

	rB, err := db.Query(qBatt, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}
	defer rB.Close()
	battery, err := scanProjBattery(rB)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}

	rT, err := db.Query(qTemp, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}
	defer rT.Close()
	outTemp, err := scanProjTemp(rT)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}

	rP, err := db.Query(qProj, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}
	defer rP.Close()
	projected, err := scanProjRange(rP)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsProjectedRangeV1", RespMetricsProjectedRange{
		Data: APIMetricsProjectedRangeData{
			Mileage:             mileage,
			BatteryLevel:        battery,
			OutdoorTemperature:  outTemp,
			ProjectedRangeCurve: projected,
			Interval:            iv,
			PreferredRange:      pr,
			WindowStartUtc:      start,
			WindowEndUtc:        end,
		},
	})
}

// TeslaMateAPICarsMetricsOverviewV1 对齐 Overview 看板核心单值/序列（组合查询）。
// @Summary 总览 KPI 包
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsOverview
// @Router /api/v1/cars/{CarID}/metrics/overview [get]
func TeslaMateAPICarsMetricsOverviewV1(c *gin.Context) {
	const errMsg = "Unable to load overview metrics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsOverviewV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsOverviewV1", "Invalid date format.", err.Error())
		return
	}
	defPR, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsOverviewV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)

	data := APIMetricsOverviewData{
		WindowStartUtc: start,
		WindowEndUtc:   end,
		PreferredRange: pr,
	}

	var bl sql.NullInt64
	_ = db.QueryRow(`
SELECT battery_level FROM (
  SELECT battery_level, date FROM positions WHERE car_id = $1 AND ideal_battery_range_km IS NOT NULL
  UNION ALL
  SELECT c.battery_level, c.date FROM charges c JOIN charging_processes p ON p.id = c.charging_process_id
  WHERE p.car_id = $1 AND c.date >= $2 AND c.date <= $3
) x ORDER BY date DESC LIMIT 1`, CarID, start, end).Scan(&bl)
	data.BatteryLevelLatest = sqlNullInt64Ptr(bl)

	var fw sql.NullString
	_ = db.QueryRow(`SELECT split_part(version, ' ', 1) FROM updates WHERE car_id = $1 ORDER BY start_date DESC LIMIT 1`, CarID).Scan(&fw)
	if fw.Valid {
		data.FirmwareVersion = fw.String
	}

	var dist sql.NullFloat64
	_ = db.QueryRow(`SELECT convert_km(sum(distance)::numeric, $2) FROM drives WHERE car_id = $1 AND start_date >= $3 AND start_date <= $4 AND end_date IS NOT NULL`,
		CarID, lenU, start, end).Scan(&dist)
	data.TotalDistanceLogged = sqlNullFloatPtr(dist)

	var net sql.NullFloat64
	netQ := fmt.Sprintf(`
SELECT sum((d.%[1]s - d.%[2]s) * c.efficiency) / convert_km(sum(d.distance)::numeric, $3) * 1000
FROM drives d JOIN cars c ON c.id = d.car_id
WHERE d.distance IS NOT NULL AND d.%[1]s - d.%[2]s >= 0.1 AND d.car_id = $4 AND d.start_date >= $5 AND d.start_date <= $6`,
		colS, colE)
	_ = db.QueryRow(netQ, lenU, CarID, start, end).Scan(&net)
	data.ConsumptionNetWhPerLength = sqlNullFloatPtr(net)

	var odo sql.NullFloat64
	_ = db.QueryRow(`SELECT convert_km(odometer::numeric, $2) FROM positions WHERE car_id = $1 AND ideal_battery_range_km IS NOT NULL ORDER BY date DESC LIMIT 1`, CarID, lenU).Scan(&odo)
	data.OdometerLatest = sqlNullFloatPtr(odo)

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsOverviewV1", RespMetricsOverview{Data: data})
}

// TeslaMateAPICarsMetricsStatesAnalyticsV1 对齐 States 看板：状态时间线数值编码、停车占比等。
// @Summary 状态分析
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsStatesAnalytics
// @Router /api/v1/cars/{CarID}/metrics/states-analytics [get]
func TeslaMateAPICarsMetricsStatesAnalyticsV1(c *gin.Context) {
	const errMsg = "Unable to load states analytics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatesAnalyticsV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatesAnalyticsV1", "Invalid date format.", err.Error())
		return
	}

	timelineQ := `
WITH s AS (
  SELECT unnest(ARRAY[start_date + interval '1 second', end_date]) AS date, unnest(ARRAY[2::numeric, 0::numeric]) AS state
  FROM charging_processes
  WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  UNION
  SELECT unnest(ARRAY[start_date + interval '1 second', end_date]), unnest(ARRAY[1::numeric, 0::numeric])
  FROM drives
  WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  UNION
  SELECT start_date, CASE WHEN state = 'offline' THEN 3 WHEN state = 'asleep' THEN 4 WHEN state = 'online' THEN 5 END
  FROM states WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
)
SELECT extract(epoch FROM date)*1000 AS t_ms, state FROM s ORDER BY date`

	rows, err := db.Query(timelineQ, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatesAnalyticsV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	tl, err := scanStatesTimeline(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsStatesAnalyticsV1", errMsg, err.Error())
		return
	}

	var parked sql.NullFloat64
	_ = db.QueryRow(`
SELECT 1 - sum(duration_min) / NULLIF((EXTRACT(EPOCH FROM (max(end_date) - min(start_date))) / 60), 0)
FROM drives WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3`, CarID, start, end).Scan(&parked)

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsStatesAnalyticsV1", RespMetricsStatesAnalytics{
		Data: APIMetricsStatesAnalyticsData{
			StateTimelineNumeric: tl,
			ParkedFraction:       sqlNullFloatPtr(parked),
			StateLegend:          "0=transition 1=drive window 2=charge window 3=offline 4=asleep 5=online（与 Grafana 编码一致）",
			WindowStartUtc:       start,
			WindowEndUtc:         end,
		},
	})
}

// TeslaMateAPICarsMetricsVisitedV1 对齐 Visited 看板统计与轨迹采样。
// @Summary Visited 看板数据
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsVisited
// @Router /api/v1/cars/{CarID}/metrics/visited [get]
func TeslaMateAPICarsMetricsVisitedV1(c *gin.Context) {
	const errMsg = "Unable to load visited metrics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVisitedV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVisitedV1", "Invalid date format.", err.Error())
		return
	}
	_, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVisitedV1", errMsg, serr.Error())
		return
	}

	var mileageStr sql.NullString
	_ = db.QueryRow(`SELECT ROUND(convert_km((max(end_km) - min(start_km))::numeric, $2),0)|| ' ' || $2
		FROM drives WHERE car_id = $1 AND start_date >= $3 AND start_date <= $4`,
		CarID, lenU, start, end).Scan(&mileageStr)

	var te, tu, eff sql.NullFloat64
	_ = db.QueryRow(`
SELECT sum(charge_energy_added), sum(greatest(charge_energy_added, charge_energy_used)),
  sum(charge_energy_added) * 100 / NULLIF(sum(greatest(charge_energy_added, charge_energy_used)), 0)
FROM charging_processes
WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3 AND charge_energy_added > 0.01`, CarID, start, end).Scan(&te, &tu, &eff)

	var tc sql.NullFloat64
	_ = db.QueryRow(`SELECT sum(cost) FROM charging_processes WHERE start_date >= $2 AND start_date <= $3 AND car_id = $1`, CarID, start, end).Scan(&tc)

	trackQ := `
SELECT extract(epoch FROM date_trunc('minute', timezone('UTC', date)))*1000 AS t_ms,
  avg(latitude) AS lat, avg(longitude) AS lon
FROM positions
WHERE car_id = $1 AND date >= $2 AND date <= $3 AND ideal_battery_range_km IS NOT NULL
GROUP BY 1 ORDER BY 1`
	rows, err := db.Query(trackQ, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVisitedV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	track, err := scanVisitedTrack(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVisitedV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsVisitedV1", RespMetricsVisited{
		Data: APIMetricsVisitedData{
			MileageLabel:          mileageStr.String,
			TotalEnergyAdded:      sqlNullFloatPtr(te),
			TotalEnergyUsed:       sqlNullFloatPtr(tu),
			ChargingEfficiencyPct: sqlNullFloatPtr(eff),
			TotalChargingCost:     sqlNullFloatPtr(tc),
			TrackSample:           track,
			WindowStartUtc:        start,
			WindowEndUtc:          end,
		},
	})
}

// TeslaMateAPICarsMetricsDutchTaxV1 对齐 reports/dutch-tax 行程表。
// @Summary 荷兰税务报表行程列表
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsDutchTax
// @Router /api/v1/cars/{CarID}/metrics/reports/dutch-tax [get]
func TeslaMateAPICarsMetricsDutchTaxV1(c *gin.Context) {
	const errMsg = "Unable to load dutch tax report."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", "Invalid date format.", err.Error())
		return
	}
	_, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", errMsg, serr.Error())
		return
	}

	q := fmt.Sprintf(`
WITH data AS (
  SELECT drives.id AS drive_id,
    floor(extract(epoch FROM start_date)) * 1000 AS start_date_ts,
    ceil(extract(epoch FROM end_date)) * 1000 AS end_date_ts,
    start_km, end_km,
    CONCAT_WS(', ', CONCAT_WS(' ', start_address.road, start_address.house_number), start_address.city) AS start_address,
    CONCAT_WS(', ', CONCAT_WS(' ', end_address.road, end_address.house_number), end_address.city) AS end_address,
    duration_min, distance
  FROM drives
  LEFT JOIN addresses start_address ON start_address_id = start_address.id
  LEFT JOIN addresses end_address ON end_address_id = end_address.id
  WHERE drives.car_id = $1 AND drives.start_date >= $2 AND drives.start_date <= $3
  ORDER BY drive_id DESC
)
SELECT drive_id, start_date_ts,
  convert_km(start_km::numeric, '%[1]s') AS start_odometer,
  start_address, end_date_ts,
  convert_km(end_km::numeric, '%[1]s') AS end_odometer,
  end_address, duration_min,
  convert_km(distance::numeric, '%[1]s') AS distance
FROM data`, lenU)

	rows, err := db.Query(q, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	list, err := scanDutchTaxDrives(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsDutchTaxV1", RespMetricsDutchTax{
		Data: APIMetricsDutchTaxData{
			Drives:         list,
			UnitOfLength:   lenU,
			WindowStartUtc: start,
			WindowEndUtc:   end,
		},
	})
}

// TeslaMateAPICarsMetricsDriveStatsExtraV1 对齐 Drive Stats：速度直方图、Top 目的地。
// @Summary 驾驶统计扩展
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Param exclude query string false "排除地址子串，逗号分隔"
// @Success 200 {object} RespMetricsDriveStatsExtra
// @Router /api/v1/cars/{CarID}/metrics/drive-stats/extra [get]
func TeslaMateAPICarsMetricsDriveStatsExtraV1(c *gin.Context) {
	const errMsg = "Unable to load drive-stats extra."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", "Invalid date format.", err.Error())
		return
	}
	_, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, serr.Error())
		return
	}

	histQ := fmt.Sprintf(`
SELECT speed_bin, SUM(seconds_elapsed) AS seconds_elapsed FROM (
SELECT
  ROUND(convert_km(p.speed::numeric, '%[1]s') / 10, 0) * 10 AS speed_bin,
  EXTRACT(EPOCH FROM (LEAD(p.date) OVER (PARTITION BY p.drive_id ORDER BY p.date) - p.date)) AS seconds_elapsed
FROM positions p
WHERE p.car_id = $1 AND p.date >= $2 AND p.date <= $3 AND p.ideal_battery_range_km IS NOT NULL
) t WHERE speed_bin > 0
GROUP BY 1 ORDER BY 1`, lenU)

	rows, err := db.Query(histQ, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	hist, err := scanSpeedHistogram(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, err.Error())
		return
	}

	destQ := `
SELECT * FROM (
SELECT COALESCE(g.name, COALESCE(a.name, nullif(CONCAT_WS(' ', a.road, a.house_number), ''))) AS name, count(*) AS visited
FROM drives t
INNER JOIN addresses a ON end_address_id = a.id
LEFT JOIN geofences g ON end_geofence_id = g.id
WHERE t.car_id = $1 AND t.start_date >= $2 AND t.start_date <= $3 AND t.end_date >= $2 AND t.end_date <= $4
GROUP BY 1
ORDER BY visited DESC) AS destinations
WHERE ($5::text = '' OR name NOT ILIKE $5)
LIMIT 10`
	exPat := "%" + strings.TrimSpace(c.Query("exclude")) + "%"
	if c.Query("exclude") == "" {
		exPat = ""
	}
	drows, err := db.Query(destQ, CarID, start, end, exPat)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, err.Error())
		return
	}
	defer drows.Close()
	dest, err := scanTopDestinations(drows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsDriveStatsExtraV1", RespMetricsDriveStatsExtra{
		Data: APIMetricsDriveStatsExtraData{
			SpeedHistogram: hist,
			TopDestinations: dest,
			WindowStartUtc: start,
			WindowEndUtc:   end,
		},
	})
}

// TeslaMateAPICarsMetricsChargingStatsExtraV1 对齐 Charging Stats：Charge Delta、Top 站点等。
// @Summary 充电统计扩展
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Param minDuration query int false "最短分钟" default(1)
// @Success 200 {object} RespMetricsChargingStatsExtra
// @Router /api/v1/cars/{CarID}/metrics/charging-stats/extra [get]
func TeslaMateAPICarsMetricsChargingStatsExtraV1(c *gin.Context) {
	const errMsg = "Unable to load charging-stats extra."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", "Invalid date format.", err.Error())
		return
	}
	minDur := convertStringToInteger(c.DefaultQuery("minDuration", "1"))

	deltaQ := `
SELECT end_date, start_battery_level, end_battery_level
FROM charging_processes
WHERE end_date >= $2 AND end_date <= $3 AND duration_min >= $4 AND car_id = $1 AND end_date IS NOT NULL
ORDER BY end_date`
	rows, err := db.Query(deltaQ, CarID, start, end, minDur)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	delta, err := scanChargeDelta(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}

	topChargedQ := `
SELECT COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS location,
  sum(charge_energy_added) AS charge_energy_added
FROM charging_processes c
LEFT JOIN addresses address ON c.address_id = address.id
LEFT JOIN geofences geofence ON geofence_id = geofence.id
WHERE end_date >= $2 AND end_date <= $3 AND duration_min >= $4 AND car_id = $1
GROUP BY 1 ORDER BY sum(charge_energy_added) DESC LIMIT 17`
	r2, err := db.Query(topChargedQ, CarID, start, end, minDur)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}
	defer r2.Close()
	topCharged, err := scanStationEnergy(r2)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}

	topCostQ := `
SELECT COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, CONCAT_WS(' ', address.road, address.house_number)), address.city)) AS location,
  sum(cost) AS cost
FROM charging_processes c
LEFT JOIN addresses address ON c.address_id = address.id
LEFT JOIN geofences geofence ON geofence_id = geofence.id
WHERE end_date >= $2 AND end_date <= $3 AND duration_min >= $4 AND car_id = $1 AND cost IS NOT NULL
GROUP BY 1 ORDER BY 2 DESC NULLS LAST LIMIT 17`
	r3, err := db.Query(topCostQ, CarID, start, end, minDur)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}
	defer r3.Close()
	topCost, err := scanStationCost(r3)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}

	geoQ := `
WITH charge_data AS (
  SELECT COALESCE(geofence.name, CONCAT_WS(', ', COALESCE(address.name, nullif(CONCAT_WS(' ', address.road, address.house_number), '')), address.city)) AS loc_nm,
    AVG(position.latitude) AS latitude,
    AVG(position.longitude) AS longitude,
    sum(charge.charge_energy_added) AS chg_total,
    count(*)::bigint AS charges
  FROM charging_processes charge
  LEFT JOIN addresses address ON charge.address_id = address.id
  LEFT JOIN positions position ON charge.position_id = position.id
  LEFT JOIN geofences geofence ON charge.geofence_id = geofence.id
  WHERE charge.end_date >= $2 AND charge.end_date <= $3
    AND charge.duration_min >= $4 AND charge.car_id = $1
  GROUP BY 1
)
SELECT loc_nm, latitude, longitude, chg_total,
  chg_total * 1.0 / NULLIF((SELECT sum(chg_total) FROM charge_data), 0) * 100 AS pct,
  charges
FROM charge_data`
	r4, err := db.Query(geoQ, CarID, start, end, minDur)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}
	defer r4.Close()
	geoByKwh, err := scanChargingGeo(r4)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsChargingStatsExtraV1", RespMetricsChargingStatsExtra{
		Data: APIMetricsChargingStatsExtraData{
			ChargeDeltaSeries:  delta,
			TopStationsByKwh:   topCharged,
			TopStationsByCost:  topCost,
			ChargingGeoByKwh:   geoByKwh,
			MinDurationMin:     minDur,
			WindowStartUtc:     start,
			WindowEndUtc:       end,
		},
	})
}

// TeslaMateAPICarsMetricsTripV1 对齐 Trip 看板关键汇总（简化，不含全部 transformation）。
// @Summary 长途 Trip 汇总
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始"
// @Param endDate query string false "结束"
// @Success 200 {object} RespMetricsTrip
// @Router /api/v1/cars/{CarID}/metrics/trip [get]
func TeslaMateAPICarsMetricsTripV1(c *gin.Context) {
	const errMsg = "Unable to load trip metrics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTripV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTripV1", "Invalid date format.", err.Error())
		return
	}
	defPR, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTripV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)

	var dist sql.NullFloat64
	_ = db.QueryRow(`
SELECT convert_km(sum(end_position.odometer - start_position.odometer)::numeric, $1) /
 NULLIF(sum(extract(epoch FROM end_position.date - start_position.date)) / 3600, 0)
FROM drives
JOIN positions start_position ON start_position_id = start_position.id
JOIN positions end_position ON end_position_id = end_position.id
WHERE drives.car_id = $2 AND drives.start_date >= $3 AND drives.start_date <= $4 AND end_date IS NOT NULL`, lenU, CarID, start, end).Scan(&dist)

	var cons sql.NullFloat64
	tripConsQ := fmt.Sprintf(`
SELECT sum((d.%[1]s - d.%[2]s) * car.efficiency * 1000) / convert_km(sum(d.distance)::numeric, $1)
FROM drives d JOIN cars car ON car.id = d.car_id
WHERE d.start_date >= $3 AND d.start_date <= $4 AND d.car_id = $2`, colS, colE)
	_ = db.QueryRow(tripConsQ, lenU, CarID, start, end).Scan(&cons)

	var cost sql.NullFloat64
	_ = db.QueryRow(`SELECT sum(cost) FROM charging_processes WHERE end_date >= $2 AND end_date <= $3 AND car_id = $1`, CarID, start, end).Scan(&cost)

	var driveHours, chargeHours sql.NullFloat64
	_ = db.QueryRow(`
SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (end_date - start_date))), 0) / 3600.0
FROM drives WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3 AND end_date IS NOT NULL`, CarID, start, end).Scan(&driveHours)
	_ = db.QueryRow(`
SELECT COALESCE(SUM(duration_min), 0) / 60.0
FROM charging_processes WHERE car_id = $1 AND end_date >= $2 AND end_date <= $3 AND end_date IS NOT NULL`, CarID, start, end).Scan(&chargeHours)

	var windowHours, parkingHoursEst sql.NullFloat64
	if t0, err := time.Parse(dbTimestampFormat, start); err == nil {
		if t1, err := time.Parse(dbTimestampFormat, end); err == nil {
			wh := t1.Sub(t0).Hours()
			windowHours = sql.NullFloat64{Float64: wh, Valid: true}
			if driveHours.Valid && chargeHours.Valid {
				ph := wh - driveHours.Float64 - chargeHours.Float64
				if ph >= 0 {
					parkingHoursEst = sql.NullFloat64{Float64: ph, Valid: true}
				}
			}
		}
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsTripV1", RespMetricsTrip{
		Data: APIMetricsTripData{
			AvgSpeedByLengthUnit:    sqlNullFloatPtr(dist),
			ConsumptionNetWhPerUnit: sqlNullFloatPtr(cons),
			TotalChargingCost:       sqlNullFloatPtr(cost),
			DriveDurationHours:      sqlNullFloatPtr(driveHours),
			ChargeDurationHours:     sqlNullFloatPtr(chargeHours),
			WindowDurationHours:     sqlNullFloatPtr(windowHours),
			ParkingHoursEst:         sqlNullFloatPtr(parkingHoursEst),
			PreferredRange:          pr,
			WindowStartUtc:          start,
			WindowEndUtc:            end,
			Note:                    "geomap 等请结合 positions；饼图可用 drive/charge/parking 时长字段。",
		},
	})
}
