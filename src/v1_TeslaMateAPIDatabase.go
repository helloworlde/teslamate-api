package main

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// TeslaMateAPIDatabaseV1 返回与 Grafana「Database info」看板类似的实例级只读信息（PostgreSQL 版本、时区等）。可选返回用户表占用与行数估算（较重）。
// @Summary 数据库实例信息
// @Description 对齐 `database-info.json` 中与 PG 实例相关的面板。`includeTableStats=true` 时查询 `pg_statio_user_tables`；`includeTableRows=true` 时对每表 count（可能很慢）。
// @Tags dashboards
// @Produce json
// @Param includeTableStats query bool false "是否包含表大小统计" default(false)
// @Param includeTableRows query bool false "是否估算各表行数（慢）" default(false)
// @Success 200 {object} RespDatabase
// @Router /api/v1/database [get]
func TeslaMateAPIDatabaseV1(c *gin.Context) {
	const errMsg = "Unable to load database info."

	includeStats := convertStringToBool(c.Query("includeTableStats"))
	includeRows := convertStringToBool(c.Query("includeTableRows"))

	info := APIDatabaseInfo{}

	var version string
	err := db.QueryRow(`SELECT regexp_replace(version(), 'PostgreSQL ([^ ]+) .*', '\1')`).Scan(&version)
	if err != nil {
		TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIDatabaseV1", errMsg, err.Error())
		return
	}
	info.PostgresVersion = version

	var tz string
	if err := db.QueryRow(`SHOW TIME ZONE`).Scan(&tz); err == nil {
		info.Timezone = &tz
	}

	var sharedBuf sql.NullString
	if err := db.QueryRow(`SELECT cast(setting AS TEXT) FROM pg_catalog.pg_settings WHERE name = 'shared_buffers'`).Scan(&sharedBuf); err == nil && sharedBuf.Valid {
		s := sharedBuf.String
		info.SharedBuffersSetting = &s
	}

	if includeStats {
		q := `SELECT relname AS table_name,
			pg_relation_size(relid) AS data_bytes,
			pg_indexes_size(relid) AS index_bytes,
			pg_total_relation_size(relid) AS total_bytes
			FROM pg_catalog.pg_statio_user_tables
			ORDER BY pg_total_relation_size(relid) DESC`
		rows, err := db.Query(q)
		if err != nil {
			TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIDatabaseV1", errMsg, err.Error())
			return
		}
		var tables []APIDatabaseTableSize
		for rows.Next() {
			var name string
			var dataB, idxB, totB int64
			if err := rows.Scan(&name, &dataB, &idxB, &totB); err != nil {
				rows.Close()
				TeslaMateAPIHandleErrorResponse(c, "TeslaMateAPIDatabaseV1", errMsg, err.Error())
				return
			}
			tables = append(tables, APIDatabaseTableSize{
				Table:      name,
				DataBytes:  dataB,
				IndexBytes: idxB,
				TotalBytes: totB,
			})
		}
		rows.Close()
		info.TableSizes = tables

		var sumTot sql.NullInt64
		if err := db.QueryRow(`SELECT COALESCE(SUM(pg_total_relation_size(relid)), 0) FROM pg_catalog.pg_statio_user_tables`).Scan(&sumTot); err == nil && sumTot.Valid {
			v := sumTot.Int64
			info.UserTablesTotalBytes = &v
		}
	}

	if includeRows {
		rows, err := db.Query(`
			SELECT table_name,
				(xpath('/row/cnt/text()', xml_count))[1]::text::bigint AS row_estimate
			FROM (
				SELECT table_name,
					query_to_xml(format('SELECT count(*) AS cnt FROM %I.%I', table_schema, table_name), false, true, '') AS xml_count
				FROM information_schema.tables
				WHERE table_schema NOT IN ('pg_catalog', 'information_schema') AND table_type = 'BASE TABLE'
			) AS t
			ORDER BY 2 DESC NULLS LAST`)
		if err != nil {
			em := err.Error()
			info.TableRowCountsError = &em
		} else {
			defer rows.Close()
			var counts []APIDatabaseTableRows
			for rows.Next() {
				var tname string
				var cnt sql.NullInt64
				if err := rows.Scan(&tname, &cnt); err != nil {
					break
				}
				row := APIDatabaseTableRows{Table: tname}
				if cnt.Valid {
					v := cnt.Int64
					row.Rows = &v
				}
				counts = append(counts, row)
			}
			info.TableRowCounts = counts
		}
	}

	TeslaMateAPIHandleSuccessResponse(c, "TeslaMateAPIDatabaseV1", RespDatabase{Data: info})
}
