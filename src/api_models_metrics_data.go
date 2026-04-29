package main

// --- GET /api/v1/database ---

// RespDatabase 数据库信息响应。
type RespDatabase struct {
	Data APIDatabaseInfo `json:"data"`
}

// APIDatabaseInfo PostgreSQL 实例与可选表统计。
type APIDatabaseInfo struct {
	PostgresVersion        string                 `json:"postgres_version"`
	Timezone               *string                `json:"timezone,omitempty"`
	SharedBuffersSetting   *string                `json:"shared_buffers_setting,omitempty"`
	TableSizes             []APIDatabaseTableSize `json:"table_sizes,omitempty"`
	UserTablesTotalBytes   *int64                 `json:"user_tables_total_bytes,omitempty"`
	TableRowCounts         []APIDatabaseTableRows `json:"table_row_counts,omitempty"`
	TableRowCountsError    *string                `json:"table_row_counts_error,omitempty"`
}

// APIDatabaseTableSize 单表占用。
type APIDatabaseTableSize struct {
	Table       string `json:"table"`
	DataBytes   int64  `json:"data_bytes"`
	IndexBytes  int64  `json:"index_bytes"`
	TotalBytes  int64  `json:"total_bytes"`
}

// APIDatabaseTableRows 单表行数估算。
type APIDatabaseTableRows struct {
	Table string `json:"table"`
	Rows  *int64 `json:"rows,omitempty"`
}

// --- metrics: charging-stats ---

// RespMetricsChargingStats 充电统计汇总。
type RespMetricsChargingStats struct {
	Data APIMetricsChargingStatsData `json:"data"`
}

// APIMetricsChargingStatsData 汇总字段（可空数值为 null）。
type APIMetricsChargingStatsData struct {
	ChargeCount                 int      `json:"charge_count"`
	TotalEnergyAddedKwh         *float64 `json:"total_energy_added_kwh"`
	TotalCost                   *float64 `json:"total_cost"`
	CostPerKwh                  *float64 `json:"cost_per_kwh"`
	WindowStartUtc              string   `json:"window_start_utc"`
	WindowEndUtc                string   `json:"window_end_utc"`
	MinDurationMin              int      `json:"min_duration_min"`
	UnitOfLengthSetting         string   `json:"unit_of_length_setting"`
	SucChargingCost             *float64 `json:"suc_charging_cost,omitempty"`
	CostPerKwhDc                *float64 `json:"cost_per_kwh_dc,omitempty"`
	CostPerKwhAc                *float64 `json:"cost_per_kwh_ac,omitempty"`
	CostPer100LengthCurrency    *float64 `json:"cost_per_100_length_currency,omitempty"`
}

// --- drive-stats ---

// RespMetricsDriveStats 驾驶统计汇总。
type RespMetricsDriveStats struct {
	Data APIMetricsDriveStatsData `json:"data"`
}

// APIMetricsDriveStatsData 驾驶汇总。
type APIMetricsDriveStatsData struct {
	DriveCount                  int      `json:"drive_count"`
	TotalDistanceKmRaw          *float64 `json:"total_distance_km_raw"`
	EnergyConsumedKwhNet        *float64 `json:"energy_consumed_kwh_net"`
	MedianDistanceConverted     *float64 `json:"median_distance_converted"`
	MaxSpeedConverted           *float64 `json:"max_speed_converted"`
	PreferredRange              string   `json:"preferred_range"`
	WindowStartUtc              string   `json:"window_start_utc"`
	WindowEndUtc                string   `json:"window_end_utc"`
	UnitOfLength                string   `json:"unit_of_length"`
	WindowDays                  *float64 `json:"window_days"`
	ExtrapolatedMonthlyMileage  *float64 `json:"extrapolated_monthly_mileage"`
	ExtrapolatedYearlyMileage   *float64 `json:"extrapolated_yearly_mileage"`
	AvgDistanceLoggedPerDay     *float64 `json:"avg_distance_logged_per_day"`
	AvgEnergyNetKwhPerDay       *float64 `json:"avg_energy_net_kwh_per_day"`
}

// --- efficiency ---

// RespMetricsEfficiency 效率指标。
type RespMetricsEfficiency struct {
	Data APIMetricsEfficiencyData `json:"data"`
}

// APIMetricsEfficiencyData 效率 KPI。
type APIMetricsEfficiencyData struct {
	ConsumptionNetWhPerUnit *float64 `json:"consumption_net_wh_per_unit"`
	LoggedDistanceConverted *float64 `json:"logged_distance_converted"`
	PreferredRange        string   `json:"preferred_range"`
	WindowStartUtc        string   `json:"window_start_utc"`
	WindowEndUtc            string   `json:"window_end_utc"`
	UnitOfLength            string   `json:"unit_of_length"`
	Note                    string   `json:"note"`
}

// --- mileage ---

// RespMetricsMileage 里程序列。
type RespMetricsMileage struct {
	Data APIMetricsMileageData `json:"data"`
}

// APIMetricsMileageData 里程序列与窗口。
type APIMetricsMileageData struct {
	Series         []APIMetricsMileagePoint `json:"series"`
	UnitOfLength   string                   `json:"unit_of_length"`
	WindowStartUtc string                   `json:"window_start_utc"`
	WindowEndUtc   string                   `json:"window_end_utc"`
}

// APIMetricsMileagePoint 单点。
type APIMetricsMileagePoint struct {
	Time    string  `json:"time"`
	Mileage float64 `json:"mileage"`
}

// --- locations ---

// RespMetricsLocations 地点聚合。
type RespMetricsLocations struct {
	Data APIMetricsLocationsData `json:"data"`
}

// APIMetricsLocationsData 地点统计。
type APIMetricsLocationsData struct {
	AddressCount       *int64                    `json:"address_count"`
	DistinctCities     *int64                    `json:"distinct_cities"`
	DistinctStates     *int64                    `json:"distinct_states"`
	DistinctCountries  *int64                    `json:"distinct_countries"`
	TopCities          []APIMetricsCityCount     `json:"top_cities"`
	AddressesSample    []APIMetricsAddressLine `json:"addresses_sample"`
	AddressFilter      string                    `json:"address_filter"`
	WindowStartUtc     string                    `json:"window_start_utc"`
	WindowEndUtc       string                    `json:"window_end_utc"`
}

// APIMetricsCityCount 城市计数。
type APIMetricsCityCount struct {
	City  string `json:"city"`
	Count int    `json:"count"`
}

// APIMetricsAddressLine 地址样本行。
type APIMetricsAddressLine struct {
	Name          string `json:"name"`
	Neighbourhood string `json:"neighbourhood"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
}

// --- timeline ---

// RespMetricsTimeline 活动时间线。
type RespMetricsTimeline struct {
	Data APIMetricsTimelineData `json:"data"`
}

// APIMetricsTimelineData 事件列表。
type APIMetricsTimelineData struct {
	Events         []APIMetricsTimelineEvent `json:"events"`
	WindowStartUtc string                    `json:"window_start_utc"`
	WindowEndUtc   string                    `json:"window_end_utc"`
	Note           string                    `json:"note"`
}

// APIMetricsTimelineEvent 时间线条目。
type APIMetricsTimelineEvent struct {
	Kind      string `json:"kind"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	RefID     string `json:"ref_id"`
	Label     string `json:"label"`
}

// --- vampire-drain ---

// RespMetricsVampireDrain 静置掉电。
type RespMetricsVampireDrain struct {
	Data APIMetricsVampireDrainData `json:"data"`
}

// APIMetricsVampireDrainData 行集与元数据。
type APIMetricsVampireDrainData struct {
	Rows             []APIMetricsVampireRow `json:"rows"`
	Columns          []string               `json:"columns"`
	DurationHoursMin float64                `json:"duration_hours_min"`
	PreferredRange   string                 `json:"preferred_range"`
	WindowStartUtc   string                 `json:"window_start_utc"`
	WindowEndUtc     string                 `json:"window_end_utc"`
}

// APIMetricsVampireRow 单条静置区间（列类型按 PG 动态，统一为 JSON 标量或 null）。
type APIMetricsVampireRow struct {
	StartDateTS       *float64 `json:"start_date_ts,omitempty"`
	EndDateTS         *float64 `json:"end_date_ts,omitempty"`
	StartDate         *string  `json:"start_date,omitempty"`
	EndDate           *string  `json:"end_date,omitempty"`
	Duration          *float64 `json:"duration,omitempty"`
	Standby           *float64 `json:"standby,omitempty"`
	SocDiff           *float64 `json:"soc_diff,omitempty"`
	HasReducedRange   *int64   `json:"has_reduced_range,omitempty"`
	RangeDiff         *float64 `json:"range_diff,omitempty"`
	Consumption       *float64 `json:"consumption,omitempty"`
	AvgPower          *float64 `json:"avg_power,omitempty"`
	RangeLostPerHour  *float64 `json:"range_lost_per_hour,omitempty"`
}

// RespAPIErrorWithHint 带排查提示的错误响应。
type RespAPIErrorWithHint struct {
	Error string `json:"error"`
	Hint  string `json:"hint"`
}

// --- statistics ---

// RespMetricsStatistics 周期统计。
type RespMetricsStatistics struct {
	Data APIMetricsStatisticsData `json:"data"`
}

// APIMetricsStatisticsData 多序列周期表。
type APIMetricsStatisticsData struct {
	DrivesPerPeriod        []APIMetricsStatisticsDrivePeriodRow    `json:"drives_per_period"`
	ChargesPerPeriod       []APIMetricsStatisticsChargePeriodRow   `json:"charges_per_period"`
	ConsumptionNetPeriod   []APIMetricsStatisticsConsumptionNetRow  `json:"consumption_net_period"`
	ConsumptionGrossPeriod []APIMetricsStatisticsConsumptionGrossRow `json:"consumption_gross_period"`
	Period                 string                                  `json:"period"`
	Timezone               string                                  `json:"timezone"`
	PreferredRange         string                                  `json:"preferred_range"`
	HighPrecision          int                                     `json:"high_precision"`
	WindowStartUtc         string                                  `json:"window_start_utc"`
	WindowEndUtc           string                                  `json:"window_end_utc"`
}

// APIMetricsStatisticsDrivePeriodRow drives 聚合行（列与 SQL 一致）。
type APIMetricsStatisticsDrivePeriodRow struct {
	DateFrom      *float64 `json:"date_from"`
	DateTo        *float64 `json:"date_to"`
	Display       string   `json:"display"`
	Date          string   `json:"date"`
	SumDurationH  *float64 `json:"sum_duration_h"`
	SumDistance   *float64 `json:"sum_distance"`
	AvgOutsideTemp *float64 `json:"avg_outside_temp"`
	Cnt        *int64   `json:"cnt"`
	Efficiency *float64 `json:"efficiency"`
}

// APIMetricsStatisticsChargePeriodRow 充电聚合行。
type APIMetricsStatisticsChargePeriodRow struct {
	DateFrom            *float64 `json:"date_from"`
	DateTo              *float64 `json:"date_to"`
	Display             string   `json:"display"`
	Date                string   `json:"date"`
	SumEnergyUsedKwh    *float64 `json:"sum_energy_used_kwh"`
	SumEnergyAddedKwh   *float64 `json:"sum_energy_added_kwh"`
	AvgEnergyChargedKwh *float64 `json:"avg_energy_charged_kwh"`
	CostCharges         *float64 `json:"cost_charges"`
	CntCharges          *float64 `json:"cnt_charges"`
}

// APIMetricsStatisticsConsumptionNetRow 净耗周期。
type APIMetricsStatisticsConsumptionNetRow struct {
	DateFrom        *float64 `json:"date_from"`
	DateTo          *float64 `json:"date_to"`
	Display         string   `json:"display"`
	Date            string   `json:"date"`
	ConsumptionNet  *float64 `json:"consumption_net"`
}

// APIMetricsStatisticsConsumptionGrossRow 毛耗周期（refId D）。
type APIMetricsStatisticsConsumptionGrossRow struct {
	DateFrom         *float64 `json:"date_from"`
	DateTo           *float64 `json:"date_to"`
	Display          string   `json:"display"`
	Date             string   `json:"date"`
	ConsumptionGross *float64 `json:"consumption_gross"`
	IsIncomplete     *bool    `json:"is_incomplete"`
}

// --- charge-level ---

// RespMetricsChargeLevel SOC 分桶序列。
type RespMetricsChargeLevel struct {
	Data APIMetricsChargeLevelData `json:"data"`
}

// APIMetricsChargeLevelThresholds SOC 阈值。
type APIMetricsChargeLevelThresholds struct {
	Lower *int64 `json:"lower"`
	Upper *int64 `json:"upper"`
}

// APIMetricsChargeLevelData 分桶序列。
type APIMetricsChargeLevelData struct {
	Series         []APIMetricsChargeLevelPoint `json:"series"`
	BucketMinutes  int                          `json:"bucket_minutes"`
	Thresholds     APIMetricsChargeLevelThresholds `json:"thresholds"`
	WindowStartUtc string                       `json:"window_start_utc"`
	WindowEndUtc   string                       `json:"window_end_utc"`
}

// APIMetricsChargeLevelPoint 分桶点。
type APIMetricsChargeLevelPoint struct {
	BucketTime         string   `json:"bucket_time"`
	BatteryLevel       *float64 `json:"battery_level"`
	UsableBatteryLevel *float64 `json:"usable_battery_level"`
}

// --- projected-range ---

// RespMetricsProjectedRange 表显续航多序列。
type RespMetricsProjectedRange struct {
	Data APIMetricsProjectedRangeData `json:"data"`
}

// APIMetricsProjectedRangeData 四序列。
type APIMetricsProjectedRangeData struct {
	Mileage              []APIMetricsProjMileageRow `json:"mileage"`
	BatteryLevel         []APIMetricsProjBattLevel  `json:"battery_level"`
	OutdoorTemperature   []APIMetricsProjTempRow    `json:"outdoor_temperature"`
	ProjectedRangeCurve  []APIMetricsProjRangeRow   `json:"projected_range_curve"`
	Interval             string                     `json:"interval"`
	PreferredRange       string                     `json:"preferred_range"`
	WindowStartUtc       string                     `json:"window_start_utc"`
	WindowEndUtc         string                     `json:"window_end_utc"`
}

// APIMetricsProjMileageRow 里程序列点。
type APIMetricsProjMileageRow struct {
	Time    string  `json:"time"`
	Mileage float64 `json:"mileage"`
}

// APIMetricsProjTempRow 外温序列点。
type APIMetricsProjTempRow struct {
	Time        string  `json:"time"`
	OutsideTemp float64 `json:"outside_temp"`
}

// APIMetricsProjRangeRow 表显续航曲线点。
type APIMetricsProjRangeRow struct {
	Time                  string  `json:"time"`
	ProjectedRangePerSoc  float64 `json:"projected_range_per_soc"`
}

// APIMetricsProjBattLevel 电量双序列点。
type APIMetricsProjBattLevel struct {
	Time               string  `json:"time"`
	BatteryLevel       float64 `json:"battery_level"`
	UsableBatteryLevel float64 `json:"usable_battery_level"`
}

// --- overview ---

// RespMetricsOverview 总览 KPI。
type RespMetricsOverview struct {
	Data APIMetricsOverviewData `json:"data"`
}

// APIMetricsOverviewData 总览字段。
type APIMetricsOverviewData struct {
	WindowStartUtc              string   `json:"window_start_utc"`
	WindowEndUtc                string   `json:"window_end_utc"`
	PreferredRange              string   `json:"preferred_range"`
	BatteryLevelLatest          *int64   `json:"battery_level_latest,omitempty"`
	FirmwareVersion             string   `json:"firmware_version,omitempty"`
	TotalDistanceLogged         *float64 `json:"total_distance_logged,omitempty"`
	ConsumptionNetWhPerLength   *float64 `json:"consumption_net_wh_per_length,omitempty"`
	OdometerLatest              *float64 `json:"odometer_latest,omitempty"`
}

// --- states-analytics ---

// RespMetricsStatesAnalytics 状态分析。
type RespMetricsStatesAnalytics struct {
	Data APIMetricsStatesAnalyticsData `json:"data"`
}

// APIMetricsStatesNumericPoint 时间线点。
type APIMetricsStatesNumericPoint struct {
	TMs   *float64 `json:"t_ms"`
	State *float64 `json:"state"`
}

// APIMetricsStatesAnalyticsData 状态分析载荷。
type APIMetricsStatesAnalyticsData struct {
	StateTimelineNumeric []APIMetricsStatesNumericPoint `json:"state_timeline_numeric"`
	ParkedFraction       *float64                       `json:"parked_fraction"`
	StateLegend          string                         `json:"state_legend"`
	WindowStartUtc       string                         `json:"window_start_utc"`
	WindowEndUtc         string                         `json:"window_end_utc"`
}

// --- visited ---

// RespMetricsVisited Visited 看板。
type RespMetricsVisited struct {
	Data APIMetricsVisitedData `json:"data"`
}

// APIMetricsVisitedData Visited 汇总。
type APIMetricsVisitedData struct {
	MileageLabel          string                        `json:"mileage_label"`
	TotalEnergyAdded      *float64                      `json:"total_energy_added"`
	TotalEnergyUsed       *float64                      `json:"total_energy_used"`
	ChargingEfficiencyPct *float64                      `json:"charging_efficiency_pct"`
	TotalChargingCost     *float64                      `json:"total_charging_cost"`
	TrackSample           []APIMetricsVisitedTrackPoint `json:"track_sample"`
	WindowStartUtc        string                        `json:"window_start_utc"`
	WindowEndUtc          string                        `json:"window_end_utc"`
}

// APIMetricsVisitedTrackPoint 轨迹采样。
type APIMetricsVisitedTrackPoint struct {
	TMs *float64 `json:"t_ms"`
	Lat *float64 `json:"lat"`
	Lon *float64 `json:"lon"`
}

// --- dutch-tax ---

// RespMetricsDutchTax 荷兰税务行程表。
type RespMetricsDutchTax struct {
	Data APIMetricsDutchTaxData `json:"data"`
}

// APIMetricsDutchTaxData 行程列表。
type APIMetricsDutchTaxData struct {
	Drives         []APIMetricsDutchTaxDrive `json:"drives"`
	UnitOfLength   string                    `json:"unit_of_length"`
	WindowStartUtc string                    `json:"window_start_utc"`
	WindowEndUtc   string                    `json:"window_end_utc"`
}

// APIMetricsDutchTaxDrive 单行。
type APIMetricsDutchTaxDrive struct {
	DriveID       int64   `json:"drive_id"`
	StartDateTs   float64 `json:"start_date_ts"`
	StartOdometer float64 `json:"start_odometer"`
	StartAddress  string  `json:"start_address"`
	EndDateTs     float64 `json:"end_date_ts"`
	EndOdometer   float64 `json:"end_odometer"`
	EndAddress    string  `json:"end_address"`
	DurationMin   int     `json:"duration_min"`
	Distance      float64 `json:"distance"`
}

// --- drive-stats/extra ---

// RespMetricsDriveStatsExtra 驾驶统计扩展。
type RespMetricsDriveStatsExtra struct {
	Data APIMetricsDriveStatsExtraData `json:"data"`
}

// APIMetricsDriveStatsExtraData 直方图与目的地。
type APIMetricsDriveStatsExtraData struct {
	SpeedHistogram []APIMetricsSpeedBin    `json:"speed_histogram"`
	TopDestinations  []APIMetricsDestCount   `json:"top_destinations"`
	WindowStartUtc   string                  `json:"window_start_utc"`
	WindowEndUtc     string                  `json:"window_end_utc"`
}

// APIMetricsSpeedBin 速度直方图桶。
type APIMetricsSpeedBin struct {
	SpeedBin        float64 `json:"speed_bin"`
	SecondsElapsed  float64 `json:"seconds_elapsed"`
}

// APIMetricsDestCount 目的地访问次数。
type APIMetricsDestCount struct {
	Name    string `json:"name"`
	Visited int    `json:"visited"`
}

// --- charging-stats/extra ---

// RespMetricsChargingStatsExtra 充电统计扩展。
type RespMetricsChargingStatsExtra struct {
	Data APIMetricsChargingStatsExtraData `json:"data"`
}

// APIMetricsChargingStatsExtraData Delta 与排行。
type APIMetricsChargingStatsExtraData struct {
	ChargeDeltaSeries  []APIMetricsChargeDeltaPoint `json:"charge_delta_series"`
	TopStationsByKwh   []APIMetricsStationEnergy    `json:"top_stations_by_kwh"`
	TopStationsByCost  []APIMetricsStationCost      `json:"top_stations_by_cost"`
	ChargingGeoByKwh   []APIMetricsChargingGeo      `json:"charging_geo_by_kwh"`
	MinDurationMin     int                          `json:"min_duration_min"`
	WindowStartUtc     string                       `json:"window_start_utc"`
	WindowEndUtc       string                       `json:"window_end_utc"`
}

// APIMetricsChargeDeltaPoint 充电 Delta 序列点。
type APIMetricsChargeDeltaPoint struct {
	EndDate           string `json:"end_date"`
	StartBatteryLevel int    `json:"start_battery_level"`
	EndBatteryLevel   int    `json:"end_battery_level"`
}

// APIMetricsStationEnergy 站点按 kWh。
type APIMetricsStationEnergy struct {
	Location          string  `json:"location"`
	ChargeEnergyAdded float64 `json:"charge_energy_added"`
}

// APIMetricsStationCost 站点按费用。
type APIMetricsStationCost struct {
	Location string  `json:"location"`
	Cost     float64 `json:"cost"`
}

// APIMetricsChargingGeo 地理加权。
type APIMetricsChargingGeo struct {
	LocNm    string   `json:"loc_nm"`
	Latitude *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	ChgTotal float64  `json:"chg_total"`
	Pct      *float64 `json:"pct"`
	Charges  int64    `json:"charges"`
}

// --- trip ---

// RespMetricsTrip Trip KPI。
type RespMetricsTrip struct {
	Data APIMetricsTripData `json:"data"`
}

// APIMetricsTripData Trip 汇总。
type APIMetricsTripData struct {
	AvgSpeedByLengthUnit     *float64 `json:"avg_speed_by_length_unit"`
	ConsumptionNetWhPerUnit  *float64 `json:"consumption_net_wh_per_unit"`
	TotalChargingCost        *float64 `json:"total_charging_cost"`
	DriveDurationHours       *float64 `json:"drive_duration_hours"`
	ChargeDurationHours      *float64 `json:"charge_duration_hours"`
	WindowDurationHours      *float64 `json:"window_duration_hours"`
	ParkingHoursEst          *float64 `json:"parking_hours_est"`
	PreferredRange           string   `json:"preferred_range"`
	WindowStartUtc           string   `json:"window_start_utc"`
	WindowEndUtc             string   `json:"window_end_utc"`
	Note                     string   `json:"note"`
}
