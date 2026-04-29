package main

import (
	"database/sql"
	"fmt"
	"time"
)

func formatStatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func projTimeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return getTimeInTimeZone(t.UTC().Format(dbTimestampFormat))
}

func scanStatisticsDrivePeriod(rows *sql.Rows) ([]APIMetricsStatisticsDrivePeriodRow, error) {
	var out []APIMetricsStatisticsDrivePeriodRow
	for rows.Next() {
		var dateFrom, dateTo sql.NullFloat64
		var display string
		var date time.Time
		var sumDurH, sumDist, avgOut, eff sql.NullFloat64
		var cnt sql.NullInt64
		if err := rows.Scan(&dateFrom, &dateTo, &display, &date, &sumDurH, &sumDist, &avgOut, &cnt, &eff); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsStatisticsDrivePeriodRow{
			DateFrom:       sqlNullFloatPtr(dateFrom),
			DateTo:         sqlNullFloatPtr(dateTo),
			Display:        display,
			Date:           formatStatDate(date),
			SumDurationH:   sqlNullFloatPtr(sumDurH),
			SumDistance:    sqlNullFloatPtr(sumDist),
			AvgOutsideTemp: sqlNullFloatPtr(avgOut),
			Cnt:            sqlNullInt64Ptr(cnt),
			Efficiency:     sqlNullFloatPtr(eff),
		})
	}
	return out, rows.Err()
}

func scanStatisticsChargePeriod(rows *sql.Rows) ([]APIMetricsStatisticsChargePeriodRow, error) {
	var out []APIMetricsStatisticsChargePeriodRow
	for rows.Next() {
		var dateFrom, dateTo sql.NullFloat64
		var display string
		var date time.Time
		var sumUsed, sumAdded, avgCh, cost sql.NullFloat64
		var cntCh sql.NullFloat64
		if err := rows.Scan(&dateFrom, &dateTo, &display, &date, &sumUsed, &sumAdded, &avgCh, &cost, &cntCh); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsStatisticsChargePeriodRow{
			DateFrom:            sqlNullFloatPtr(dateFrom),
			DateTo:              sqlNullFloatPtr(dateTo),
			Display:             display,
			Date:                formatStatDate(date),
			SumEnergyUsedKwh:    sqlNullFloatPtr(sumUsed),
			SumEnergyAddedKwh:   sqlNullFloatPtr(sumAdded),
			AvgEnergyChargedKwh: sqlNullFloatPtr(avgCh),
			CostCharges:         sqlNullFloatPtr(cost),
			CntCharges:          sqlNullFloatPtr(cntCh),
		})
	}
	return out, rows.Err()
}

func scanStatisticsConsumptionNet(rows *sql.Rows) ([]APIMetricsStatisticsConsumptionNetRow, error) {
	var out []APIMetricsStatisticsConsumptionNetRow
	for rows.Next() {
		var dateFrom, dateTo sql.NullFloat64
		var display string
		var date time.Time
		var cons sql.NullFloat64
		if err := rows.Scan(&dateFrom, &dateTo, &display, &date, &cons); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsStatisticsConsumptionNetRow{
			DateFrom:       sqlNullFloatPtr(dateFrom),
			DateTo:         sqlNullFloatPtr(dateTo),
			Display:        display,
			Date:           formatStatDate(date),
			ConsumptionNet: sqlNullFloatPtr(cons),
		})
	}
	return out, rows.Err()
}

func scanStatisticsConsumptionGross(rows *sql.Rows) ([]APIMetricsStatisticsConsumptionGrossRow, error) {
	var out []APIMetricsStatisticsConsumptionGrossRow
	for rows.Next() {
		var dateFrom, dateTo sql.NullFloat64
		var display string
		var date time.Time
		var gross sql.NullFloat64
		var incomp sql.NullBool
		if err := rows.Scan(&dateFrom, &dateTo, &display, &date, &gross, &incomp); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsStatisticsConsumptionGrossRow{
			DateFrom:         sqlNullFloatPtr(dateFrom),
			DateTo:           sqlNullFloatPtr(dateTo),
			Display:          display,
			Date:             formatStatDate(date),
			ConsumptionGross: sqlNullFloatPtr(gross),
			IsIncomplete:     sqlNullBoolPtr(incomp),
		})
	}
	return out, rows.Err()
}

func sqlNullBoolPtr(n sql.NullBool) *bool {
	if !n.Valid {
		return nil
	}
	b := n.Bool
	return &b
}

func scanChargeLevelSeries(rows *sql.Rows) ([]APIMetricsChargeLevelPoint, error) {
	var out []APIMetricsChargeLevelPoint
	for rows.Next() {
		var bt time.Time
		var bl, ubl sql.NullFloat64
		if err := rows.Scan(&bt, &bl, &ubl); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsChargeLevelPoint{
			BucketTime:         projTimeString(bt),
			BatteryLevel:       sqlNullFloatPtr(bl),
			UsableBatteryLevel: sqlNullFloatPtr(ubl),
		})
	}
	return out, rows.Err()
}

func scanProjMileage(rows *sql.Rows) ([]APIMetricsProjMileageRow, error) {
	var out []APIMetricsProjMileageRow
	for rows.Next() {
		var t time.Time
		var m float64
		if err := rows.Scan(&t, &m); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsProjMileageRow{Time: projTimeString(t), Mileage: m})
	}
	return out, rows.Err()
}

func scanProjBattery(rows *sql.Rows) ([]APIMetricsProjBattLevel, error) {
	var out []APIMetricsProjBattLevel
	for rows.Next() {
		var t time.Time
		var bl, ubl float64
		if err := rows.Scan(&t, &bl, &ubl); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsProjBattLevel{Time: projTimeString(t), BatteryLevel: bl, UsableBatteryLevel: ubl})
	}
	return out, rows.Err()
}

func scanProjTemp(rows *sql.Rows) ([]APIMetricsProjTempRow, error) {
	var out []APIMetricsProjTempRow
	for rows.Next() {
		var t time.Time
		var tp float64
		if err := rows.Scan(&t, &tp); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsProjTempRow{Time: projTimeString(t), OutsideTemp: tp})
	}
	return out, rows.Err()
}

func scanProjRange(rows *sql.Rows) ([]APIMetricsProjRangeRow, error) {
	var out []APIMetricsProjRangeRow
	for rows.Next() {
		var t time.Time
		var pr float64
		if err := rows.Scan(&t, &pr); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsProjRangeRow{Time: projTimeString(t), ProjectedRangePerSoc: pr})
	}
	return out, rows.Err()
}

func scanStatesTimeline(rows *sql.Rows) ([]APIMetricsStatesNumericPoint, error) {
	var out []APIMetricsStatesNumericPoint
	for rows.Next() {
		var tms, st sql.NullFloat64
		if err := rows.Scan(&tms, &st); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsStatesNumericPoint{
			TMs:   sqlNullFloatPtr(tms),
			State: sqlNullFloatPtr(st),
		})
	}
	return out, rows.Err()
}

func scanVisitedTrack(rows *sql.Rows) ([]APIMetricsVisitedTrackPoint, error) {
	var out []APIMetricsVisitedTrackPoint
	for rows.Next() {
		var tms sql.NullFloat64
		var lat, lon sql.NullFloat64
		if err := rows.Scan(&tms, &lat, &lon); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsVisitedTrackPoint{
			TMs: sqlNullFloatPtr(tms),
			Lat: sqlNullFloatPtr(lat),
			Lon: sqlNullFloatPtr(lon),
		})
	}
	return out, rows.Err()
}

func scanDutchTaxDrives(rows *sql.Rows) ([]APIMetricsDutchTaxDrive, error) {
	var out []APIMetricsDutchTaxDrive
	for rows.Next() {
		var d APIMetricsDutchTaxDrive
		if err := rows.Scan(&d.DriveID, &d.StartDateTs, &d.StartOdometer, &d.StartAddress, &d.EndDateTs, &d.EndOdometer, &d.EndAddress, &d.DurationMin, &d.Distance); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func scanSpeedHistogram(rows *sql.Rows) ([]APIMetricsSpeedBin, error) {
	var out []APIMetricsSpeedBin
	for rows.Next() {
		var b APIMetricsSpeedBin
		if err := rows.Scan(&b.SpeedBin, &b.SecondsElapsed); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func scanTopDestinations(rows *sql.Rows) ([]APIMetricsDestCount, error) {
	var out []APIMetricsDestCount
	for rows.Next() {
		var name string
		var n int
		if err := rows.Scan(&name, &n); err != nil {
			return nil, err
		}
		out = append(out, APIMetricsDestCount{Name: name, Visited: n})
	}
	return out, rows.Err()
}

func scanChargeDelta(rows *sql.Rows) ([]APIMetricsChargeDeltaPoint, error) {
	var out []APIMetricsChargeDeltaPoint
	for rows.Next() {
		var end time.Time
		var sb, eb int
		if err := rows.Scan(&end, &sb, &eb); err != nil {
			return nil, err
		}
		endStr := end.UTC().Format(dbTimestampFormat)
		out = append(out, APIMetricsChargeDeltaPoint{
			EndDate:           getTimeInTimeZone(endStr),
			StartBatteryLevel: sb,
			EndBatteryLevel:   eb,
		})
	}
	return out, rows.Err()
}

func scanStationEnergy(rows *sql.Rows) ([]APIMetricsStationEnergy, error) {
	var out []APIMetricsStationEnergy
	for rows.Next() {
		var r APIMetricsStationEnergy
		if err := rows.Scan(&r.Location, &r.ChargeEnergyAdded); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanStationCost(rows *sql.Rows) ([]APIMetricsStationCost, error) {
	var out []APIMetricsStationCost
	for rows.Next() {
		var r APIMetricsStationCost
		if err := rows.Scan(&r.Location, &r.Cost); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanChargingGeo(rows *sql.Rows) ([]APIMetricsChargingGeo, error) {
	var out []APIMetricsChargingGeo
	for rows.Next() {
		var r APIMetricsChargingGeo
		var lat, lon sql.NullFloat64
		var pct sql.NullFloat64
		if err := rows.Scan(&r.LocNm, &lat, &lon, &r.ChgTotal, &pct, &r.Charges); err != nil {
			return nil, err
		}
		r.Latitude = sqlNullFloatPtr(lat)
		r.Longitude = sqlNullFloatPtr(lon)
		r.Pct = sqlNullFloatPtr(pct)
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanVampireRows(rows *sql.Rows) ([]APIMetricsVampireRow, []string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	var out []APIMetricsVampireRow
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, cols, err
		}
		out = append(out, vampireRowFromValues(cols, vals))
	}
	return out, cols, rows.Err()
}

func vampireRowFromValues(cols []string, vals []interface{}) APIMetricsVampireRow {
	var r APIMetricsVampireRow
	for i, c := range cols {
		if i >= len(vals) {
			break
		}
		switch c {
		case "start_date_ts":
			r.StartDateTS = anyToFloatPtr(vals[i])
		case "end_date_ts":
			r.EndDateTS = anyToFloatPtr(vals[i])
		case "start_date":
			r.StartDate = anyToTimeStringPtr(vals[i])
		case "end_date":
			r.EndDate = anyToTimeStringPtr(vals[i])
		case "duration":
			r.Duration = anyToFloatPtr(vals[i])
		case "standby":
			r.Standby = anyToFloatPtr(vals[i])
		case "soc_diff":
			r.SocDiff = anyToFloatPtr(vals[i])
		case "has_reduced_range":
			r.HasReducedRange = anyToInt64Ptr(vals[i])
		case "range_diff":
			r.RangeDiff = anyToFloatPtr(vals[i])
		case "consumption":
			r.Consumption = anyToFloatPtr(vals[i])
		case "avg_power":
			r.AvgPower = anyToFloatPtr(vals[i])
		case "range_lost_per_hour":
			r.RangeLostPerHour = anyToFloatPtr(vals[i])
		}
	}
	return r
}

func anyToFloatPtr(v interface{}) *float64 {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case float64:
		return &x
	case float32:
		f := float64(x)
		return &f
	case int64:
		f := float64(x)
		return &f
	case int32:
		f := float64(x)
		return &f
	case int:
		f := float64(x)
		return &f
	case []uint8:
		var f float64
		if _, err := fmt.Sscanf(string(x), "%f", &f); err == nil {
			return &f
		}
		return nil
	default:
		return nil
	}
}

func anyToInt64Ptr(v interface{}) *int64 {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case int64:
		return &x
	case int32:
		i := int64(x)
		return &i
	case int:
		i := int64(x)
		return &i
	case bool:
		var i int64
		if x {
			i = 1
		}
		return &i
	case float64:
		i := int64(x)
		return &i
	default:
		return nil
	}
}

func anyToTimeStringPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case time.Time:
		s := formatStatDate(x)
		return &s
	case []uint8:
		s := string(x)
		return &s
	case string:
		return &x
	default:
		s := fmt.Sprint(x)
		return &s
	}
}
