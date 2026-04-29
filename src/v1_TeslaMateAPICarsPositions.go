package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsPositionsV1 返回 `positions` 采样点，对应 Overview、Locations、Projected range 等看板中的轨迹与曲线数据源。
// @Summary 车辆位置与采样点
// @Description 查询 `positions` 表（时间序列与经纬度等）。支持 startDate、endDate 过滤 `date` 列，分页参数 page、show（单页最大 5000）。
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间 RFC3339"
// @Param endDate query string false "结束时间 RFC3339"
// @Param page query int false "页码，从 1 开始" default(1)
// @Param show query int false "每页条数" default(500)
// @Success 200 {object} RespPositionsList
// @Router /api/v1/cars/{CarID}/positions [get]
func TeslaMateAPICarsPositionsV1(c *gin.Context) {
	const errMsg = "Unable to load positions."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", errMsg, "invalid CarID")
		return
	}

	parsedStart, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", "Invalid date format.", err.Error())
		return
	}
	parsedEnd, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", "Invalid date format.", err.Error())
		return
	}

	page := convertStringToInteger(c.DefaultQuery("page", "1"))
	show := convertStringToInteger(c.DefaultQuery("show", "500"))
	if page > 0 {
		page--
	} else {
		page = 0
	}
	if show <= 0 {
		show = 500
	}
	if show > 5000 {
		show = 5000
	}
	offset := page * show

	query := `
		SELECT
			positions.id,
			positions.date,
			positions.latitude,
			positions.longitude,
			positions.odometer,
			COALESCE(positions.ideal_battery_range_km, 0),
			COALESCE(positions.rated_battery_range_km, 0),
			positions.battery_level,
			positions.usable_battery_level,
			positions.speed,
			positions.power,
			positions.outside_temp,
			positions.inside_temp,
			positions.driver_temp_setting
		FROM positions
		WHERE positions.car_id = $1`

	var args []any
	args = append(args, CarID)
	pi := 2
	if parsedStart != "" {
		query += fmt.Sprintf(" AND positions.date >= $%d", pi)
		args = append(args, parsedStart)
		pi++
	}
	if parsedEnd != "" {
		query += fmt.Sprintf(" AND positions.date <= $%d", pi)
		args = append(args, parsedEnd)
		pi++
	}
	query += fmt.Sprintf(` ORDER BY positions.date DESC LIMIT $%d OFFSET $%d`, pi, pi+1)
	args = append(args, show, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", errMsg, err.Error())
		return
	}
	defer rows.Close()

	var (
		unitsLen, unitsTemp string
		list                []APIPositionRow
	)
	_ = db.QueryRow(`SELECT unit_of_length, unit_of_temperature FROM settings LIMIT 1`).Scan(&unitsLen, &unitsTemp)

	for rows.Next() {
		var p APIPositionRow
		err := rows.Scan(
			&p.ID,
			&p.Date,
			&p.Latitude,
			&p.Longitude,
			&p.Odometer,
			&p.IdealBatteryRangeKM,
			&p.RatedBatteryRangeKM,
			&p.BatteryLevel,
			&p.UsableBatteryLevel,
			&p.Speed,
			&p.Power,
			&p.OutsideTemp,
			&p.InsideTemp,
			&p.DriverTempSetting,
		)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", errMsg, err.Error())
			return
		}
		p.Date = getTimeInTimeZone(p.Date)
		if unitsLen == "mi" {
			p.Odometer = kilometersToMiles(p.Odometer)
			p.IdealBatteryRangeKM = kilometersToMiles(p.IdealBatteryRangeKM)
			p.RatedBatteryRangeKM = kilometersToMiles(p.RatedBatteryRangeKM)
		}
		if unitsTemp == "F" {
			p.OutsideTemp = celsiusToFahrenheitNilSupport(p.OutsideTemp)
			p.InsideTemp = celsiusToFahrenheitNilSupport(p.InsideTemp)
			p.DriverTempSetting = celsiusToFahrenheitNilSupport(p.DriverTempSetting)
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsPositionsV1", errMsg, err.Error())
		return
	}

	var carName NullString
	_ = db.QueryRow(`SELECT name FROM cars WHERE id = $1`, CarID).Scan(&carName)

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsPositionsV1", RespPositionsList{
		Data: RespPositionsData{
			Car: APICarRef{
				CarID:   CarID,
				CarName: carName,
			},
			Positions: list,
			TeslaMateUnits: APIUnitsLengthTemp{
				UnitsLength:      unitsLen,
				UnitsTemperature: unitsTemp,
			},
		},
	})
}
