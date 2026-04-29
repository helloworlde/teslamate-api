package main

// 与 Grafana 看板 JSON（dashboards/*.json）中对应面板的 rawSql 对齐；运行环境 PostgreSQL 16+。

import (
	"fmt"
	"strings"
)

func escapeSQLIdentTZ(tz string) string {
	return strings.ReplaceAll(tz, "'", "''")
}

// chargingStatsCostPer100Query 对齐 charging-stats「Ø Cost per 100 $length_unit」面板（窗口时长在 SQL 内分支 48h）。
func chargingStatsCostPer100Query(pr string) string {
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)
	br := fmt.Sprintf("%s_battery_range_km", pr)
	secDiff := `(EXTRACT(EPOCH FROM ($3::timestamptz - $2::timestamptz)))`
	tfDrive := `start_date >= $2 AND start_date <= $3`
	tfDriveSub := `car_id = $1 AND start_date >= $2 AND start_date <= $3`
	tfCpEnd := `end_date >= $2 AND end_date <= $3`
	tfPos := `p.date >= $2 AND p.date <= $3`
	lenU := `(SELECT unit_of_length FROM settings LIMIT 1)`

	return fmt.Sprintf(`
WITH drives_start_event AS (
  SELECT 'drive_start' AS event, start_date AS date, %[1]s AS range, start_km AS odometer, car_id, distance IS NULL AS is_incomplete
  FROM drives
  WHERE car_id = $1 AND %[2]s AND 48 <= (%[3]s::numeric / 3600)
),
drives_end_event AS (
  SELECT 'drive_end' AS event, CASE WHEN end_date IS NULL THEN start_date + interval '1 second' ELSE end_date END AS date,
    %[4]s AS range, end_km AS odometer, car_id, distance IS NULL AS is_incomplete
  FROM drives
  WHERE car_id = $1 AND %[2]s AND 48 <= (%[3]s::numeric / 3600)
),
charging_processes_start_event AS (
  SELECT 'charging_process_start' AS event, cp.start_date AS date, %[1]s AS range, p.odometer, cp.car_id, cp.end_date IS NULL AS is_incomplete
  FROM charging_processes cp INNER JOIN positions p ON cp.position_id = p.id
  WHERE cp.car_id = $1 AND cp.end_date >= $2 AND cp.end_date <= $3 AND 48 <= (%[3]s::numeric / 3600)
),
charging_processes_end_event AS (
  SELECT 'charging_process_end' AS event, CASE WHEN cp.end_date IS NULL THEN cp.start_date + interval '1 second' ELSE cp.end_date END AS date,
    %[4]s AS range, p.odometer, cp.car_id, cp.end_date IS NULL AS is_incomplete
  FROM charging_processes cp INNER JOIN positions p ON cp.position_id = p.id
  WHERE cp.car_id = $1 AND %[5]s AND 48 <= (%[3]s::numeric / 3600)
),
positions AS (
  SELECT CASE WHEN drive_id IS NOT NULL AND lead(drive_id) OVER w IS NOT NULL THEN 'drive_start' ELSE 'something' END AS event,
    p.date, %[6]s AS range, p.odometer, p.car_id, false AS is_incomplete
  FROM positions p
  WHERE p.ideal_battery_range_km IS NOT NULL AND p.car_id = $1 AND 48 > (%[3]s::numeric / 3600)
    AND (p.drive_id IN (SELECT id FROM drives WHERE %[7]s) OR (p.drive_id IS NULL AND %[8]s))
  WINDOW w AS (ORDER BY p.date)
),
combined AS (
  SELECT * FROM drives_start_event UNION ALL SELECT * FROM drives_end_event UNION ALL
  SELECT * FROM charging_processes_start_event UNION ALL SELECT * FROM charging_processes_end_event UNION ALL SELECT * FROM positions
),
final AS (
  SELECT car_id,
    CASE WHEN is_incomplete THEN 0 ELSE lead(odometer) OVER w - odometer END AS distance,
    CASE WHEN is_incomplete THEN 0 ELSE CASE WHEN event != 'drive_start' THEN greatest(range - lead(range) OVER w, 0) ELSE range - lead(range) OVER w END END AS range_loss
  FROM combined WINDOW w AS (ORDER BY date ASC)
),
derived AS (
  SELECT convert_km(sum(distance)::numeric, %[9]s) AS distance, sum(range_loss) * c.efficiency AS consumption
  FROM final INNER JOIN cars c ON car_id = c.id GROUP BY c.efficiency
),
charges AS (
  SELECT sum(cost) / sum(charge_energy_added) AS cost_per_kwh
  FROM charging_processes WHERE car_id = $1 AND %[10]s
)
SELECT consumption / distance * 100 * cost_per_kwh AS cost_mileage
FROM derived CROSS JOIN charges`,
		colS, tfDrive, secDiff, colE, tfCpEnd, br, tfDriveSub, tfPos, lenU, tfCpEnd)
}

// statisticsRefIDGrossQuery 对齐 statistics.json refId D（高精度毛电耗按 period 聚合）。highPrec: 0=事件流，1=positions 细粒度。
func statisticsRefIDGrossQuery(pr, period, tz string, hp int) string {
	colS := fmt.Sprintf("start_%s_range_km", pr)
	colE := fmt.Sprintf("end_%s_range_km", pr)
	br := fmt.Sprintf("%s_battery_range_km", pr)
	qtzLit := "'" + escapeSQLIdentTZ(tz) + "'"
	tfDrive := `start_date >= $2 AND start_date <= $3`
	tfCp := `start_date >= $2 AND start_date <= $3`
	tfDrivesSub := `car_id = $1 AND start_date >= $2 AND start_date <= $3`
	tfPosDate := `p.date >= $2 AND p.date <= $3`
	lenU := `(SELECT unit_of_length FROM settings LIMIT 1)`
	ival := fmt.Sprintf(`interval '1 %s'`, period)

	return fmt.Sprintf(`
WITH drives_start_event AS (
  SELECT 'drive_start' AS event, start_date AS date, %[1]s AS range, start_km AS odometer, car_id, distance IS NULL AS is_incomplete
  FROM drives
  WHERE car_id = $1 AND %[2]s AND 0 = %[10]d
),
drives_end_event AS (
  SELECT 'drive_end' AS event, CASE WHEN end_date IS NULL THEN start_date + interval '1 second' ELSE end_date END AS date,
    %[3]s AS range, end_km AS odometer, car_id, distance IS NULL AS is_incomplete
  FROM drives
  WHERE car_id = $1 AND %[2]s AND 0 = %[10]d
),
charging_processes_start_event AS (
  SELECT 'charging_process_start' AS event, cp.start_date AS date, %[1]s AS range, p.odometer, cp.car_id, cp.end_date IS NULL AS is_incomplete
  FROM charging_processes cp INNER JOIN positions p ON cp.position_id = p.id
  WHERE cp.car_id = $1 AND %[4]s AND 0 = %[10]d
),
charging_processes_end_event AS (
  SELECT 'charging_process_end' AS event, CASE WHEN cp.end_date IS NULL THEN cp.start_date + interval '1 second' ELSE cp.end_date END AS date,
    %[3]s AS range, p.odometer, cp.car_id, cp.end_date IS NULL AS is_incomplete
  FROM charging_processes cp INNER JOIN positions p ON cp.position_id = p.id
  WHERE cp.car_id = $1 AND %[4]s AND 0 = %[10]d
),
positions AS (
  SELECT CASE WHEN drive_id IS NOT NULL AND lead(drive_id) OVER w IS NOT NULL THEN 'drive_start' ELSE 'something' END AS event,
    p.date, %[5]s AS range, p.odometer, p.car_id, false AS is_incomplete
  FROM positions p
  WHERE p.ideal_battery_range_km IS NOT NULL AND p.car_id = $1 AND 1 = %[10]d
    AND (p.drive_id IN (SELECT id FROM drives WHERE %[6]s) OR (p.drive_id IS NULL AND %[7]s))
  WINDOW w AS (ORDER BY p.date)
),
combined AS (
  SELECT * FROM drives_start_event UNION ALL SELECT * FROM drives_end_event UNION ALL
  SELECT * FROM charging_processes_start_event UNION ALL SELECT * FROM charging_processes_end_event UNION ALL SELECT * FROM positions
),
final AS (
  SELECT car_id,
    date_trunc('%[8]s', timezone('UTC', date), %[9]s) AS date,
    CASE WHEN is_incomplete THEN 0 ELSE lead(odometer) OVER w - odometer END AS distance,
    CASE WHEN is_incomplete THEN 0 ELSE CASE WHEN event != 'drive_start' THEN greatest(range - lead(range) OVER w, 0) ELSE range - lead(range) OVER w END END AS range_loss,
    sum(CASE WHEN is_incomplete THEN 1 ELSE 0 END) OVER w > 0 AS is_incomplete
  FROM combined WINDOW w AS (ORDER BY date asc)
)
SELECT
  EXTRACT(EPOCH FROM date) * 1000 AS date_from,
  EXTRACT(EPOCH FROM date + %[11]s) * 1000 AS date_to,
  CASE '%[8]s'
    WHEN 'month' THEN to_char(timezone(%[9]s, date), 'YYYY Month')
    WHEN 'year' THEN to_char(timezone(%[9]s, date), 'YYYY')
    WHEN 'week' THEN 'week ' || to_char(timezone(%[9]s, date), 'WW') || ' starting ' || to_char(timezone(%[9]s, date), 'YYYY-MM-DD')
    ELSE to_char(timezone(%[9]s, date), 'YYYY-MM-DD')
  END AS display,
  date,
  (sum(range_loss) * c.efficiency * 1000) / NULLIF(convert_km(sum(distance)::numeric, %[12]s), 0) AS consumption_gross,
  is_incomplete
FROM final INNER JOIN cars c ON car_id = c.id
GROUP BY 1, 2, 3, 4, c.efficiency, is_incomplete`,
		colS, tfDrive, colE, tfCp, br, tfDrivesSub, tfPosDate,
		period, qtzLit, hp, ival, lenU)
}
