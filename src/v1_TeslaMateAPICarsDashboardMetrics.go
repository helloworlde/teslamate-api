package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsMetricsChargingStatsV1 对齐 Charging Stats 看板顶部汇总（次数、能量、费用等）。
// @Summary 充电统计汇总
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始（RFC3339），缺省为最近 30 天起点"
// @Param endDate query string false "结束（RFC3339），缺省为当前 UTC"
// @Param minDuration query int false "最短充电时长（分钟），对应 min_duration" default(1)
// @Param preferredRange query string false "ideal|rated，用于 Ø Cost per 100 等派生"
// @Success 200 {object} RespMetricsChargingStats
// @Router /api/v1/cars/{CarID}/metrics/charging-stats [get]
func TeslaMateAPICarsMetricsChargingStatsV1(c *gin.Context) {
	const errMsg = "Unable to load charging stats."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsV1", "Invalid date format.", err.Error())
		return
	}
	minDur := convertStringToInteger(c.DefaultQuery("minDuration", "1"))
	if minDur < 0 {
		minDur = 1
	}

	q := `
WITH base AS (
  SELECT * FROM charging_processes cp
  WHERE cp.car_id = $1 AND cp.end_date IS NOT NULL
    AND cp.end_date >= $2 AND cp.end_date <= $3 AND cp.duration_min >= $4
)
SELECT
  (SELECT COUNT(*) FROM base) AS charge_count,
  (SELECT COALESCE(SUM(charge_energy_added), 0) FROM base) AS total_energy_added_kwh,
  (SELECT COALESCE(SUM(cost), 0) FROM base) AS total_cost,
  (SELECT COALESCE(SUM(cost) / NULLIF(SUM(GREATEST(charge_energy_added, charge_energy_used)), 0), 0) FROM base) AS cost_per_kwh,
  (SELECT unit_of_length FROM settings LIMIT 1) AS length_unit
`
	var chargeCount int
	var totalEnergy, totalCost, costPerKwh sql.NullFloat64
	var lengthUnit string
	err = db.QueryRow(q, CarID, start, end, minDur).Scan(&chargeCount, &totalEnergy, &totalCost, &costPerKwh, &lengthUnit)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsChargingStatsV1", errMsg, err.Error())
		return
	}

	data := APIMetricsChargingStatsData{
		ChargeCount:            chargeCount,
		TotalEnergyAddedKwh:    sqlNullFloatPtr(totalEnergy),
		TotalCost:              sqlNullFloatPtr(totalCost),
		CostPerKwh:             sqlNullFloatPtr(costPerKwh),
		WindowStartUtc:         start,
		WindowEndUtc:           end,
		MinDurationMin:         minDur,
		UnitOfLengthSetting:    lengthUnit,
	}

	sucQ := `
SELECT COALESCE(SUM(cp.cost), 0)
FROM charging_processes cp
LEFT JOIN addresses addr ON addr.id = cp.address_id
LEFT JOIN geofences geo ON geo.id = cp.geofence_id
JOIN charges ch ON ch.charging_process_id = cp.id AND ch.date = cp.end_date
WHERE cp.end_date >= $2 AND cp.end_date <= $3
  AND cp.duration_min >= $4 AND cp.car_id = $1 AND cp.end_date IS NOT NULL
  AND (addr.name ILIKE '%supercharger%' OR geo.name ILIKE '%supercharger%' OR ch.fast_charger_brand = 'Tesla')
  AND NULLIF(ch.charger_phases, 0) IS NULL
  AND ch.fast_charger_type != 'ACSingleWireCAN'
  AND cp.cost IS NOT NULL`
	var sucCost sql.NullFloat64
	if err := db.QueryRow(sucQ, CarID, start, end, minDur).Scan(&sucCost); err == nil {
		data.SucChargingCost = sqlNullFloatPtr(sucCost)
	}

	dcACBase := `
WITH data AS (
  SELECT cp.id, cp.cost, cp.charge_energy_added, cp.charge_energy_used,
    CASE WHEN NULLIF(mode() WITHIN GROUP (ORDER BY ch.charger_phases), 0) IS NULL THEN 'DC' ELSE 'AC' END AS current
  FROM charging_processes cp
  RIGHT JOIN charges ch ON cp.id = ch.charging_process_id
  WHERE cp.car_id = $1 AND cp.duration_min >= $4 AND cp.end_date >= $2 AND cp.end_date <= $3 AND cp.end_date IS NOT NULL
  GROUP BY 1, 2, 3, 4
)
SELECT COALESCE(SUM(cost) / NULLIF(SUM(GREATEST(charge_energy_added, charge_energy_used)), 0), 0) FROM data WHERE current = $5`
	var dcCostPerKwh, acCostPerKwh sql.NullFloat64
	if err := db.QueryRow(dcACBase, CarID, start, end, minDur, "DC").Scan(&dcCostPerKwh); err == nil {
		data.CostPerKwhDc = sqlNullFloatPtr(dcCostPerKwh)
	}
	if err := db.QueryRow(dcACBase, CarID, start, end, minDur, "AC").Scan(&acCostPerKwh); err == nil {
		data.CostPerKwhAc = sqlNullFloatPtr(acCostPerKwh)
	}

	if defPR, _, _, lerr := loadDashboardSettings(); lerr == nil {
		pr := preferredRangeFromQuery(c, defPR)
		if _, verr := validatePreferredRangeColumn(pr); verr == nil {
			costQ := chargingStatsCostPer100Query(pr)
			var costPer100 sql.NullFloat64
			if err := db.QueryRow(costQ, CarID, start, end).Scan(&costPer100); err == nil {
				data.CostPer100LengthCurrency = sqlNullFloatPtr(costPer100)
			}
		}
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsChargingStatsV1", RespMetricsChargingStats{Data: data})
}

// TeslaMateAPICarsMetricsDriveStatsV1 对齐 Drive Stats 看板汇总指标。
// @Summary 驾驶统计汇总
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param preferredRange query string false "ideal 或 rated，默认 settings"
// @Success 200 {object} RespMetricsDriveStats
// @Router /api/v1/cars/{CarID}/metrics/drive-stats [get]
func TeslaMateAPICarsMetricsDriveStatsV1(c *gin.Context) {
	const errMsg = "Unable to load drive stats."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", "Invalid date format.", err.Error())
		return
	}
	defPR, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	if _, verr := validatePreferredRangeColumn(pr); verr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", verr.Error(), verr.Error())
		return
	}
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)

	q := fmt.Sprintf(`
WITH since AS (
  SELECT timezone('UTC', min(start_date)) AS s FROM drives WHERE car_id = $3
),
bounds AS (
  SELECT GREATEST($1::timestamptz, COALESCE((SELECT s FROM since), $1::timestamptz)) AS wfrom,
         $2::timestamptz AS wto
),
wd AS (
  SELECT GREATEST(EXTRACT(EPOCH FROM ((SELECT wto FROM bounds) - (SELECT wfrom FROM bounds))) / 86400.0, 1.0) AS days
),
odo AS (
  SELECT COALESCE(MAX(end_km) - MIN(start_km), 0) AS span_km
  FROM drives
  WHERE car_id = $3 AND start_date >= $1 AND start_date <= $2 AND end_date IS NOT NULL
),
agg AS (
  SELECT
    COUNT(*) FILTER (WHERE d.end_date IS NOT NULL) AS drive_count,
    COALESCE(SUM(d.distance) FILTER (WHERE d.end_date IS NOT NULL), 0) AS sum_distance_km,
    COALESCE(SUM((d.%[1]s - d.%[2]s) * c.efficiency) FILTER (WHERE d.end_date IS NOT NULL), 0) AS energy_consumed_kwh_net,
    convert_km((percentile_cont(0.5) WITHIN GROUP (ORDER BY d.distance))::numeric, $4) AS median_distance,
    convert_km(MAX(d.speed_max)::numeric, $4) AS max_speed
  FROM drives d
  INNER JOIN cars c ON c.id = d.car_id
  WHERE d.car_id = $3 AND d.start_date >= $1 AND d.start_date <= $2
)
SELECT
  agg.drive_count,
  agg.sum_distance_km,
  agg.energy_consumed_kwh_net,
  agg.median_distance,
  agg.max_speed,
  $4 AS length_unit,
  wd.days AS window_days,
  convert_km((odo.span_km / wd.days * (365.0 / 12.0))::numeric, $4) AS extrapolated_monthly,
  convert_km((odo.span_km / wd.days * 365.0)::numeric, $4) AS extrapolated_yearly,
  convert_km((agg.sum_distance_km / wd.days)::numeric, $4) AS avg_distance_logged_per_day,
  (agg.energy_consumed_kwh_net / wd.days) AS avg_energy_net_kwh_per_day
FROM agg, wd, odo
`, colS, colE)

	var driveCount int
	var sumDist, energyNet, medianDist, maxSp sql.NullFloat64
	var lengthUnit string
	var windowDays sql.NullFloat64
	var extrapMo, extrapYr, avgDistDay sql.NullFloat64
	var avgEnergyDay sql.NullFloat64
	err = db.QueryRow(q, start, end, CarID, lenU).Scan(&driveCount, &sumDist, &energyNet, &medianDist, &maxSp, &lengthUnit,
		&windowDays, &extrapMo, &extrapYr, &avgDistDay, &avgEnergyDay)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsDriveStatsV1", RespMetricsDriveStats{
		Data: APIMetricsDriveStatsData{
			DriveCount:                 driveCount,
			TotalDistanceKmRaw:         sqlNullFloatPtr(sumDist),
			EnergyConsumedKwhNet:       sqlNullFloatPtr(energyNet),
			MedianDistanceConverted:    sqlNullFloatPtr(medianDist),
			MaxSpeedConverted:          sqlNullFloatPtr(maxSp),
			PreferredRange:             pr,
			WindowStartUtc:             start,
			WindowEndUtc:               end,
			UnitOfLength:               lengthUnit,
			WindowDays:                 sqlNullFloatPtr(windowDays),
			ExtrapolatedMonthlyMileage: sqlNullFloatPtr(extrapMo),
			ExtrapolatedYearlyMileage:  sqlNullFloatPtr(extrapYr),
			AvgDistanceLoggedPerDay:    sqlNullFloatPtr(avgDistDay),
			AvgEnergyNetKwhPerDay:      sqlNullFloatPtr(avgEnergyDay),
		},
	})
}

// TeslaMateAPICarsMetricsEfficiencyV1 对齐 Efficiency 看板核心 KPI（净/毛电耗、记录里程）。
// @Summary 效率指标
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param preferredRange query string false "ideal 或 rated"
// @Success 200 {object} RespMetricsEfficiency
// @Router /api/v1/cars/{CarID}/metrics/efficiency [get]
func TeslaMateAPICarsMetricsEfficiencyV1(c *gin.Context) {
	const errMsg = "Unable to load efficiency metrics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", "Invalid date format.", err.Error())
		return
	}
	defPR, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	if _, verr := validatePreferredRangeColumn(pr); verr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", verr.Error(), verr.Error())
		return
	}
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)

	netQ := fmt.Sprintf(`
SELECT
  sum((d.%[1]s - d.%[2]s) * c.efficiency) / convert_km(sum(d.distance)::numeric, $1) * 1000
FROM drives d
INNER JOIN cars c ON c.id = d.car_id
WHERE d.distance IS NOT NULL
  AND d.%[1]s - d.%[2]s >= 0.1
  AND d.start_date >= $2 AND d.start_date <= $3
  AND d.car_id = $4
`, colS, colE)

	grossQ := `
SELECT convert_km(sum(d.distance)::numeric, $1)
FROM drives d
WHERE d.car_id = $4 AND d.start_date >= $2 AND d.start_date <= $3 AND d.end_date IS NOT NULL`

	var netConsumption sql.NullFloat64
	var loggedDist sql.NullFloat64
	err = db.QueryRow(netQ, lenU, start, end, CarID).Scan(&netConsumption)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", errMsg, err.Error())
		return
	}
	err = db.QueryRow(grossQ, lenU, start, end, CarID).Scan(&loggedDist)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsEfficiencyV1", RespMetricsEfficiency{
		Data: APIMetricsEfficiencyData{
			ConsumptionNetWhPerUnit: sqlNullFloatPtr(netConsumption),
			LoggedDistanceConverted: sqlNullFloatPtr(loggedDist),
			PreferredRange:          pr,
			WindowStartUtc:          start,
			WindowEndUtc:            end,
			UnitOfLength:            lenU,
			Note:                    "consumption_net 与 Grafana 一致：Wh/单位长度（由 convert_km 换算）",
		},
	})
}

// TeslaMateAPICarsMetricsMileageV1 里程时间序列（Mileage 看板）。
// @Summary 里程曲线数据
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Success 200 {object} RespMetricsMileage
// @Router /api/v1/cars/{CarID}/metrics/mileage [get]
func TeslaMateAPICarsMetricsMileageV1(c *gin.Context) {
	const errMsg = "Unable to load mileage series."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsMileageV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsMileageV1", "Invalid date format.", err.Error())
		return
	}
	_, lenU, _, lerr := loadDashboardSettings()
	if lerr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsMileageV1", errMsg, lerr.Error())
		return
	}

	q := `
WITH o AS (
  SELECT start_date AS time, car_id, start_km AS odometer FROM drives WHERE car_id = $3
  UNION ALL
  SELECT end_date, car_id, end_km AS odometer FROM drives WHERE car_id = $3
)
SELECT time, convert_km(odometer::numeric, $4) AS mileage
FROM o
WHERE car_id = $3 AND time >= $1 AND time <= $2
ORDER BY 1`
	rows, err := db.Query(q, start, end, CarID, lenU)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsMileageV1", errMsg, err.Error())
		return
	}
	defer rows.Close()

	var series []APIMetricsMileagePoint
	for rows.Next() {
		var t string
		var m float64
		if err := rows.Scan(&t, &m); err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsMileageV1", errMsg, err.Error())
			return
		}
		series = append(series, APIMetricsMileagePoint{Time: getTimeInTimeZone(t), Mileage: m})
	}
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsMileageV1", RespMetricsMileage{
		Data: APIMetricsMileageData{
			Series:         series,
			UnitOfLength:   lenU,
			WindowStartUtc: start,
			WindowEndUtc:   end,
		},
	})
}

// TeslaMateAPICarsMetricsLocationsV1 地点统计（Locations 看板：地址数、城市榜等）。
// @Summary 地点与地址聚合
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param addressFilter query string false "地址模糊过滤，对应 address_filter"
// @Success 200 {object} RespMetricsLocations
// @Router /api/v1/cars/{CarID}/metrics/locations [get]
func TeslaMateAPICarsMetricsLocationsV1(c *gin.Context) {
	const errMsg = "Unable to load locations metrics."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsLocationsV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsLocationsV1", "Invalid date format.", err.Error())
		return
	}
	filter := "%" + c.Query("addressFilter") + "%"

	statQ := `
SELECT count(*), count(DISTINCT city), count(DISTINCT state), count(DISTINCT country)
FROM addresses WHERE id IN (
  SELECT start_address_id FROM drives WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  UNION
  SELECT end_address_id FROM drives WHERE car_id = $1 AND end_date >= $2 AND end_date <= $3
  UNION
  SELECT address_id FROM charging_processes WHERE car_id = $1
    AND ((start_date >= $2 AND start_date <= $3) OR (end_date >= $2 AND end_date <= $3))
)`
	var nAddr, nCity, nState, nCountry sql.NullInt64
	err = db.QueryRow(statQ, CarID, start, end).Scan(&nAddr, &nCity, &nState, &nCountry)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsLocationsV1", errMsg, err.Error())
		return
	}

	citiesQ := `
SELECT city, count(*) AS n FROM addresses
WHERE city IS NOT NULL AND id IN (
  SELECT start_address_id FROM drives WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  UNION SELECT end_address_id FROM drives WHERE car_id = $1 AND end_date >= $2 AND end_date <= $3
  UNION SELECT address_id FROM charging_processes WHERE car_id = $1
    AND ((start_date >= $2 AND start_date <= $3) OR (end_date >= $2 AND end_date <= $3))
)
GROUP BY 1 ORDER BY 2 DESC LIMIT 10`
	rows, err := db.Query(citiesQ, CarID, start, end)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsLocationsV1", errMsg, err.Error())
		return
	}
	defer rows.Close()
	var cities []APIMetricsCityCount
	for rows.Next() {
		var city string
		var n int
		if err := rows.Scan(&city, &n); err != nil {
			break
		}
		cities = append(cities, APIMetricsCityCount{City: city, Count: n})
	}

	addrQ := `
SELECT COALESCE(name, CONCAT(road, ' ', house_number)), neighbourhood, city, state, country
FROM addresses
WHERE display_name ILIKE $4 AND id IN (
  SELECT start_address_id FROM drives WHERE car_id = $1 AND start_date >= $2 AND start_date <= $3
  UNION SELECT end_address_id FROM drives WHERE car_id = $1 AND end_date >= $2 AND end_date <= $3
  UNION SELECT address_id FROM charging_processes WHERE car_id = $1
    AND ((start_date >= $2 AND start_date <= $3) OR (end_date >= $2 AND end_date <= $3))
)
ORDER BY inserted_at DESC LIMIT 100`
	arows, err := db.Query(addrQ, CarID, start, end, filter)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsLocationsV1", errMsg, err.Error())
		return
	}
	defer arows.Close()
	var addresses []APIMetricsAddressLine
	for arows.Next() {
		var name, nb, city, state, country sql.NullString
		if err := arows.Scan(&name, &nb, &city, &state, &country); err != nil {
			break
		}
		addresses = append(addresses, APIMetricsAddressLine{
			Name:          strings.TrimSpace(name.String),
			Neighbourhood: nb.String,
			City:          city.String,
			State:         state.String,
			Country:       country.String,
		})
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsLocationsV1", RespMetricsLocations{
		Data: APIMetricsLocationsData{
			AddressCount:       sqlNullInt64Ptr(nAddr),
			DistinctCities:     sqlNullInt64Ptr(nCity),
			DistinctStates:     sqlNullInt64Ptr(nState),
			DistinctCountries:  sqlNullInt64Ptr(nCountry),
			TopCities:          cities,
			AddressesSample:    addresses,
			AddressFilter:      c.Query("addressFilter"),
			WindowStartUtc:     start,
			WindowEndUtc:       end,
		},
	})
}

// TeslaMateAPICarsMetricsTimelineV1 简化时间线：驾驶、充电、软件更新（Timeline 看板的子集，不含停车推算）。
// @Summary 活动时间线（简化）
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param actions query string false "逗号分隔：driving,charging,updating；缺省为全部"
// @Param textFilter query string false "模糊匹配（版本号等）"
// @Success 200 {object} RespMetricsTimeline
// @Router /api/v1/cars/{CarID}/metrics/timeline [get]
func TeslaMateAPICarsMetricsTimelineV1(c *gin.Context) {
	const errMsg = "Unable to load timeline."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTimelineV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTimelineV1", "Invalid date format.", err.Error())
		return
	}
	actions := c.Query("actions")
	want := map[string]bool{}
	for _, a := range strings.Split(strings.ToLower(actions), ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			want[a] = true
		}
	}
	defaultAll := len(want) == 0
	tf := "%" + c.Query("textFilter") + "%"

	q := `
SELECT 'driving' AS kind, d.start_date::text, d.end_date::text, d.id::text AS ref,
  COALESCE(sg.name, CONCAT_WS(', ', COALESCE(sa.name, nullif(CONCAT_WS(' ', sa.road, sa.house_number), '')), sa.city)) AS label
FROM drives d
LEFT JOIN addresses sa ON d.start_address_id = sa.id
LEFT JOIN geofences sg ON d.start_geofence_id = sg.id
WHERE d.car_id = $1 AND d.start_date >= $2 AND d.start_date <= $3
  AND ($4 = '' OR COALESCE(sg.name, CONCAT_WS(', ', COALESCE(sa.name, nullif(CONCAT_WS(' ', sa.road, sa.house_number), '')), sa.city)) ILIKE $4)
UNION ALL
SELECT 'charging', cp.start_date::text, cp.end_date::text, cp.id::text,
  COALESCE(g.name, CONCAT_WS(', ', COALESCE(a.name, nullif(CONCAT_WS(' ', a.road, a.house_number), '')), a.city))
FROM charging_processes cp
LEFT JOIN addresses a ON cp.address_id = a.id
LEFT JOIN geofences g ON cp.geofence_id = g.id
WHERE cp.car_id = $1 AND cp.start_date >= $2 AND cp.start_date <= $3
  AND ($4 = '' OR COALESCE(g.name, CONCAT_WS(', ', COALESCE(a.name, nullif(CONCAT_WS(' ', a.road, a.house_number), '')), a.city)) ILIKE $4)
UNION ALL
SELECT 'updating', u.start_date::text, u.end_date::text, u.id::text, split_part(u.version, ' ', 1)
FROM updates u
WHERE u.car_id = $1 AND u.start_date >= $2 AND u.start_date <= $3
  AND ($4 = '' OR u.version ILIKE $4)
ORDER BY 2 DESC
LIMIT 800`
	tfArg := tf
	if c.Query("textFilter") == "" {
		tfArg = ""
	}
	rows, err := db.Query(q, CarID, start, end, tfArg)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsTimelineV1", errMsg, err.Error())
		return
	}
	defer rows.Close()

	var events []APIMetricsTimelineEvent
	for rows.Next() {
		var kind, s, e, ref, label string
		if err := rows.Scan(&kind, &s, &e, &ref, &label); err != nil {
			break
		}
		events = append(events, APIMetricsTimelineEvent{
			Kind:      kind,
			StartDate: getTimeInTimeZone(s),
			EndDate:   getTimeInTimeZone(e),
			RefID:     ref,
			Label:     label,
		})
	}

	var filtered []APIMetricsTimelineEvent
	for _, ev := range events {
		k := strings.ToLower(ev.Kind)
		if !defaultAll && !want[k] {
			continue
		}
		filtered = append(filtered, ev)
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsTimelineV1", RespMetricsTimeline{
		Data: APIMetricsTimelineData{
			Events:         filtered,
			WindowStartUtc: start,
			WindowEndUtc:   end,
			Note:           "不含 Grafana Timeline 中的停车/缺失推算；完整逻辑见看板 SQL。",
		},
	})
}

// TeslaMateAPICarsMetricsVampireDrainV1 吸血鬼放电表（Vampire Drain 看板），依赖 TeslaMate DB 函数 convert_km。
// @Summary 静置掉电（吸血鬼放电）
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间"
// @Param endDate query string false "结束时间"
// @Param preferredRange query string false "ideal 或 rated"
// @Param durationHours query number false "最小间隔（小时），对应 duration 变量" default(6)
// @Success 200 {object} RespMetricsVampireDrain
// @Router /api/v1/cars/{CarID}/metrics/vampire-drain [get]
func TeslaMateAPICarsMetricsVampireDrainV1(c *gin.Context) {
	const errMsg = "Unable to load vampire drain."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", errMsg, "invalid CarID")
		return
	}
	start, end, err := effectiveMetricsTimeRangeString(c)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", "Invalid date format.", err.Error())
		return
	}
	defPR, lenU, _, serr := loadDashboardSettings()
	if serr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", errMsg, serr.Error())
		return
	}
	pr := preferredRangeFromQuery(c, defPR)
	if _, verr := validatePreferredRangeColumn(pr); verr != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", verr.Error(), verr.Error())
		return
	}
	durH := convertStringToFloat(c.DefaultQuery("durationHours", "6"))
	if durH <= 0 {
		durH = 6
	}

	colER := fmt.Sprintf("end_%s_range_km", pr)
	colSR := fmt.Sprintf("start_%s_range_km", pr)

	sqlStr := fmt.Sprintf(`
WITH merge AS (
 SELECT c.start_date, c.end_date, c.start_ideal_range_km, c.end_ideal_range_km, c.start_rated_range_km, c.end_rated_range_km,
    c.start_battery_level, c.end_battery_level, p.usable_battery_level AS start_usable_battery_level, NULL::smallint AS end_usable_battery_level,
    p.odometer AS start_km, p.odometer AS end_km
 FROM charging_processes c JOIN positions p ON c.position_id = p.id
 WHERE c.car_id = $1 AND c.start_date >= $2 AND c.start_date <= $3
 UNION
 SELECT d.start_date, d.end_date, d.start_ideal_range_km, d.end_ideal_range_km, d.start_rated_range_km, d.end_rated_range_km,
    sp.battery_level, ep.battery_level, sp.usable_battery_level, ep.usable_battery_level, d.start_km, d.end_km
 FROM drives d
 JOIN positions sp ON d.start_position_id = sp.id JOIN positions ep ON d.end_position_id = ep.id
 WHERE d.car_id = $1 AND d.start_date >= $2 AND d.start_date <= $3
),
v AS (
 SELECT
    lag(t.end_date) OVER w AS start_date,
    t.start_date AS end_date,
    lag(t.%[1]s) OVER w AS start_range,
    t.%[2]s AS end_range,
    lag(t.end_km) OVER w AS start_km,
    t.start_km AS end_km,
    EXTRACT(EPOCH FROM age(t.start_date, lag(t.end_date) OVER w)) AS duration,
    lag(t.end_battery_level) OVER w AS start_battery_level,
    lag(t.end_usable_battery_level) OVER w AS start_usable_battery_level,
    t.start_battery_level AS end_battery_level,
    t.start_usable_battery_level AS end_usable_battery_level,
    t.start_battery_level > COALESCE(t.start_usable_battery_level, t.start_battery_level) AS has_reduced_range
 FROM merge t
 WINDOW w AS (ORDER BY t.start_date ASC)
)
SELECT
  floor(extract(epoch FROM v.start_date)) * 1000 AS start_date_ts,
  ceil(extract(epoch FROM v.end_date)) * 1000 AS end_date_ts,
  v.start_date, v.end_date, v.duration,
  (coalesce(s_asleep.sleep, 0) + coalesce(s_offline.sleep, 0)) / NULLIF(v.duration, 0) AS standby,
  -greatest(v.start_battery_level - v.end_battery_level, 0) AS soc_diff,
  CASE WHEN v.has_reduced_range THEN 1 ELSE 0 END AS has_reduced_range,
  convert_km(CASE WHEN v.has_reduced_range THEN NULL ELSE (v.start_range - v.end_range)::numeric END, $5) AS range_diff,
  CASE WHEN v.has_reduced_range THEN NULL ELSE (v.start_range - v.end_range) * c.efficiency END AS consumption,
  CASE WHEN v.has_reduced_range THEN NULL ELSE ((v.start_range - v.end_range) * c.efficiency) / (v.duration / 3600) * 1000 END AS avg_power,
  convert_km(CASE WHEN v.has_reduced_range THEN NULL ELSE ((v.start_range - v.end_range) / (v.duration / 3600))::numeric END, $5) AS range_lost_per_hour
FROM v
LEFT JOIN LATERAL (
  SELECT EXTRACT(EPOCH FROM sum(age(s.end_date, s.start_date))) AS sleep
  FROM states s
  WHERE state = 'asleep' AND v.start_date IS NOT NULL AND v.start_date <= s.start_date AND s.end_date <= v.end_date AND s.car_id = $1
) s_asleep ON true
LEFT JOIN LATERAL (
  SELECT EXTRACT(EPOCH FROM sum(age(s.end_date, s.start_date))) AS sleep
  FROM states s
  WHERE state = 'offline' AND v.start_date IS NOT NULL AND v.start_date <= s.start_date AND s.end_date <= v.end_date AND s.car_id = $1
) s_offline ON true
JOIN cars c ON c.id = $1
WHERE v.duration > ($4 * 3600)
  AND v.start_range - v.end_range >= 0
  AND v.end_km - v.start_km < 1
`, colER, colSR)

	rows, err := db.Query(sqlStr, CarID, start, end, durH, lenU)
	if err != nil {
		TeslaMateAPIHandleOtherResponse(c, http.StatusInternalServerError, "TeslaMateAPICarsMetricsVampireDrainV1", RespAPIErrorWithHint{
			Error: err.Error(),
			Hint:  "若失败请确认 TeslaMate 已安装 convert_km 等 DB 函数，且 states 表结构匹配。",
		})
		return
	}
	defer rows.Close()
	out, cols, err := scanVampireRows(rows)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", errMsg, err.Error())
		return
	}
	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsMetricsVampireDrainV1", RespMetricsVampireDrain{
		Data: APIMetricsVampireDrainData{
			Rows:             out,
			Columns:          cols,
			DurationHoursMin: durH,
			PreferredRange:   pr,
			WindowStartUtc:   start,
			WindowEndUtc:     end,
		},
	})
}
