package main

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPICarsStatesV1 返回 `states` 表记录，对应 Grafana「States」看板及 Overview 状态相关面板。
// @Summary 车辆在线状态历史
// @Description 查询 TeslaMate `states` 表（offline / asleep / online）。支持 startDate、endDate（RFC3339），默认按 start_date 降序。
// @Tags dashboards
// @Produce json
// @Param CarID path int true "车辆 ID" example(1)
// @Param startDate query string false "开始时间 RFC3339"
// @Param endDate query string false "结束时间 RFC3339"
// @Param page query int false "页码，从 1 开始" default(1)
// @Param show query int false "每页条数" default(100)
// @Success 200 {object} RespStatesList
// @Router /api/v1/cars/{CarID}/states [get]
func TeslaMateAPICarsStatesV1(c *gin.Context) {
	const errMsg = "Unable to load states."
	CarID := convertStringToInteger(c.Param("CarID"))
	if CarID == 0 {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", errMsg, "invalid CarID")
		return
	}

	parsedStart, err := parseDateParam(c.Query("startDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", "Invalid date format.", err.Error())
		return
	}
	parsedEnd, err := parseDateParam(c.Query("endDate"))
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", "Invalid date format.", err.Error())
		return
	}

	page := convertStringToInteger(c.DefaultQuery("page", "1"))
	show := convertStringToInteger(c.DefaultQuery("show", "100"))
	if page > 0 {
		page--
	} else {
		page = 0
	}
	if show <= 0 || show > 5000 {
		show = 100
	}
	offset := page * show

	query := `SELECT id, state, start_date, end_date FROM states WHERE car_id = $1`
	var args []any
	args = append(args, CarID)
	pi := 2
	if parsedStart != "" {
		query += fmt.Sprintf(" AND start_date >= $%d", pi)
		args = append(args, parsedStart)
		pi++
	}
	if parsedEnd != "" {
		query += fmt.Sprintf(" AND start_date <= $%d", pi)
		args = append(args, parsedEnd)
		pi++
	}
	query += fmt.Sprintf(` ORDER BY start_date DESC LIMIT $%d OFFSET $%d`, pi, pi+1)
	args = append(args, show, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", errMsg, err.Error())
		return
	}
	defer rows.Close()

	var name NullString
	_ = db.QueryRow(`SELECT name FROM cars WHERE id = $1`, CarID).Scan(&name)

	var list []APIStateRow
	for rows.Next() {
		var (
			id        int
			state     string
			startDate string
			endDate   sql.NullString
		)
		if err := rows.Scan(&id, &state, &startDate, &endDate); err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", errMsg, err.Error())
			return
		}
		row := APIStateRow{ID: id, State: state, StartDate: getTimeInTimeZone(startDate)}
		if endDate.Valid {
			row.EndDate = getTimeInTimeZone(endDate.String)
		}
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPICarsStatesV1", errMsg, err.Error())
		return
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPICarsStatesV1", RespStatesList{
		Data: RespStatesData{
			Car: APICarRef{
				CarID:   CarID,
				CarName: name,
			},
			States: list,
		},
	})
}
