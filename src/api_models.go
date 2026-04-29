package main

// --- 通用 / 错误 / 系统 ---

// RespAPIError 通用错误 JSON（如 401/403/501 或业务错误载荷）。
type RespAPIError struct {
	// Error 错误说明
	Error string `json:"error"`
}

// RespPageNotFound 未注册路由。
type RespPageNotFound struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RespHealthz 存活探针响应。
type RespHealthz struct {
	// Status HTTP 状态文本
	Status string `json:"status"`
}

// RespReadyz 就绪探针成功响应。
type RespReadyz struct {
	// Status HTTP 状态文本
	Status string `json:"status"`
}

// RespPing Ping 响应。
type RespPing struct {
	// Message 固定为 pong
	Message string `json:"message"`
}

// RespHTTPRoot 根路径提示。
type RespHTTPRoot struct {
	// Message 运行提示
	Message string `json:"message"`
	// Path 当前路径
	Path string `json:"path"`
}

// --- 多接口复用：车辆引用与单位 ---

// APICarRef 响应中标识车辆（car_id + 可选名称）。
type APICarRef struct {
	// CarID TeslaMate cars.id
	CarID int `json:"car_id"`
	// CarName 车辆昵称，可为空
	CarName NullString `json:"car_name"`
}

// APIUnitsLengthTemp settings 中的长度与温度单位。
type APIUnitsLengthTemp struct {
	// UnitsLength km 或 mi
	UnitsLength string `json:"unit_of_length"`
	// UnitsTemperature C 或 F
	UnitsTemperature string `json:"unit_of_temperature"`
}

// APIUnitsLengthTempPressure status 接口增加胎压单位。
type APIUnitsLengthTempPressure struct {
	// UnitsLength km 或 mi
	UnitsLength string `json:"unit_of_length"`
	// UnitsPressure bar 或 psi
	UnitsPressure string `json:"unit_of_pressure"`
	// UnitsTemperature C 或 F
	UnitsTemperature string `json:"unit_of_temperature"`
}

// --- GET /api/v1/cars ---

// RespCarsList GET /api/v1/cars 与 /api/v1/cars/:CarID 成功响应。
type RespCarsList struct {
	// Data 车辆列表或单车
	Data RespCarsListData `json:"data"`
}

// RespCarsListData data 载荷。
type RespCarsListData struct {
	// Cars 车辆数组
	Cars []APICarsListCar `json:"cars"`
}

// APICarsListCar 列表中单辆车。
type APICarsListCar struct {
	// CarID 车辆 ID
	CarID int `json:"car_id"`
	// Name 昵称
	Name NullString `json:"name"`
	// CarDetails 车型与效率等
	CarDetails APICarsListCarDetails `json:"car_details"`
	// CarExterior 外观
	CarExterior APICarsListCarExterior `json:"car_exterior"`
	// CarSettings 车辆设置
	CarSettings APICarsListCarSettings `json:"car_settings"`
	// TeslaMateDetails 写入/更新时间
	TeslaMateDetails APICarsListTMDetails `json:"teslamate_details"`
	// TeslaMateStats 充电/驾驶/更新次数统计
	TeslaMateStats APICarsListTMStats `json:"teslamate_stats"`
}

// APICarsListCarDetails 数据库 cars 与效率字段。
type APICarsListCarDetails struct {
	// EID Tesla 车辆 entity id
	EID int64 `json:"eid"`
	// VID 车辆 id
	VID int64 `json:"vid"`
	// Vin 车架号
	Vin string `json:"vin"`
	// Model 车型字符串
	Model NullString `json:"model"`
	// TrimBadging 尾标
	TrimBadging NullString `json:"trim_badging"`
	// Efficiency 效率系数
	Efficiency NullFloat64 `json:"efficiency"`
}

// APICarsListCarExterior 外观配置。
type APICarsListCarExterior struct {
	// ExteriorColor 颜色
	ExteriorColor string `json:"exterior_color"`
	// SpoilerType 尾翼
	SpoilerType string `json:"spoiler_type"`
	// WheelType 轮毂
	WheelType string `json:"wheel_type"`
}

// APICarsListCarSettings car_settings 表字段。
type APICarsListCarSettings struct {
	// SuspendMin 挂起分钟
	SuspendMin int `json:"suspend_min"`
	// SuspendAfterIdleMin 空闲后挂起
	SuspendAfterIdleMin int `json:"suspend_after_idle_min"`
	// ReqNotUnlocked 要求未解锁
	ReqNotUnlocked bool `json:"req_not_unlocked"`
	// FreeSupercharging 免费超充
	FreeSupercharging bool `json:"free_supercharging"`
	// UseStreamingAPI 流式 API
	UseStreamingAPI bool `json:"use_streaming_api"`
}

// APICarsListTMDetails TeslaMate 记录时间。
type APICarsListTMDetails struct {
	// InsertedAt 创建时间
	InsertedAt string `json:"inserted_at"`
	// UpdatedAt 更新时间
	UpdatedAt string `json:"updated_at"`
}

// APICarsListTMStats 聚合计数。
type APICarsListTMStats struct {
	// TotalCharges 充电次数
	TotalCharges int `json:"total_charges"`
	// TotalDrives 驾驶次数
	TotalDrives int `json:"total_drives"`
	// TotalUpdates 更新次数
	TotalUpdates int `json:"total_updates"`
}

// --- charges 列表 ---

// RespChargesList GET .../charges 响应。
type RespChargesList struct {
	Data RespChargesListData `json:"data"`
}

// RespChargesListData 充电列表 data。
type RespChargesListData struct {
	Car            APICarRef          `json:"car"`
	Charges        []APIChargesRow    `json:"charges"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIChargesBatteryLevels 起止 SOC。
type APIChargesBatteryLevels struct {
	StartBatteryLevel int `json:"start_battery_level"`
	EndBatteryLevel   int `json:"end_battery_level"`
}

// APIChargesRangeSpan ideal/rated 起止续航（km 或 mi，依设置换算）。
type APIChargesRangeSpan struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
}

// APIChargesRow 单次充电摘要行。
type APIChargesRow struct {
	ChargeID          int                   `json:"charge_id"`
	StartDate         string                `json:"start_date"`
	EndDate           string                `json:"end_date"`
	Address           string                `json:"address"`
	ChargeEnergyAdded float64               `json:"charge_energy_added"`
	ChargeEnergyUsed  float64               `json:"charge_energy_used"`
	Cost              float64               `json:"cost"`
	DurationMin       int                   `json:"duration_min"`
	DurationStr       string                `json:"duration_str"`
	BatteryDetails    APIChargesBatteryLevels `json:"battery_details"`
	RangeIdeal        APIChargesRangeSpan   `json:"range_ideal"`
	RangeRated        APIChargesRangeSpan   `json:"range_rated"`
	OutsideTempAvg    float64               `json:"outside_temp_avg"`
	Odometer          float64               `json:"odometer"`
	Latitude          float64               `json:"latitude"`
	Longitude         float64               `json:"longitude"`
}

// --- charges/current ---

// RespChargeCurrent GET .../charges/current 响应。
type RespChargeCurrent struct {
	Data RespChargeCurrentData `json:"data"`
}

// RespChargeCurrentData 当前充电 data。
type RespChargeCurrentData struct {
	Car            APICarRef          `json:"car"`
	Charge         APIChargeCurrent   `json:"charge"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIChargeCurrentBattery 起止与当前 SOC。
type APIChargeCurrentBattery struct {
	StartBatteryLevel   int `json:"start_battery_level"`
	CurrentBatteryLevel int `json:"current_battery_level"`
}

// APIChargeCurrentRatedRange rated 续航区间与增量。
type APIChargeCurrentRatedRange struct {
	StartRange   float64 `json:"start_range"`
	CurrentRange float64 `json:"current_range"`
	AddedRange   float64 `json:"added_range"`
}

// APIChargeChargerStats 充电桩瞬时读数。
type APIChargeChargerStats struct {
	ChargerActualCurrent int `json:"charger_actual_current"`
	ChargerPhases        int `json:"charger_phases"`
	ChargerPilotCurrent  int `json:"charger_pilot_current"`
	ChargerPower         int `json:"charger_power"`
	ChargerVoltage       int `json:"charger_voltage"`
}

// APIChargeFastCharger 快充信息。
type APIChargeFastCharger struct {
	FastChargerPresent bool    `json:"fast_charger_present"`
	FastChargerBrand   *string `json:"fast_charger_brand,omitempty"`
	FastChargerType    *string `json:"fast_charger_type,omitempty"`
}

// APIChargeDetailBatteryInfo 采样点电池信息。
type APIChargeDetailBatteryInfo struct {
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power"`
}

// APIChargeDetailRow 充电过程内采样点（current 与 details 共用形状差异见 json）。
type APIChargeDetailRow struct {
	DetailID             int                 `json:"detail_id"`
	Date                 string              `json:"date"`
	BatteryLevel         int                 `json:"battery_level"`
	UsableBatteryLevel   int                 `json:"usable_battery_level"`
	ChargeEnergyAdded    float64             `json:"charge_energy_added"`
	NotEnoughPowerToHeat NullBool            `json:"not_enough_power_to_heat"`
	ChargerDetails       APIChargeChargerStats `json:"charger_details"`
	BatteryInfo          APIChargeDetailBatteryInfo `json:"battery_info"`
	ConnChargeCable      interface{}         `json:"conn_charge_cable,omitempty"`
	FastChargerInfo      APIChargeFastCharger `json:"fast_charger_info"`
	OutsideTemp          float64             `json:"outside_temp"`
}

// APIChargeCurrent 当前充电汇总。
type APIChargeCurrent struct {
	ChargeID          int                       `json:"charge_id"`
	StartDate         string                    `json:"start_date"`
	IsCharging        bool                      `json:"is_charging"`
	Address           string                    `json:"address"`
	ChargeEnergyAdded float64                   `json:"charge_energy_added"`
	Cost              float64                   `json:"cost"`
	DurationMin       int                       `json:"duration_min"`
	DurationStr       string                    `json:"duration_str"`
	BatteryDetails    APIChargeCurrentBattery   `json:"battery_details"`
	RatedRange        APIChargeCurrentRatedRange `json:"rated_range"`
	OutsideTempAvg    float64                   `json:"outside_temp_avg"`
	Odometer          float64                   `json:"odometer"`
	ChargeDetails     []APIChargeDetailRow      `json:"charge_details"`
}

// --- charges/:ChargeID 详情 ---

// RespChargeDetail GET .../charges/:ChargeID 响应。
type RespChargeDetail struct {
	Data RespChargeDetailData `json:"data"`
}

// RespChargeDetailData 充电详情 data。
type RespChargeDetailData struct {
	Car            APICarRef          `json:"car"`
	Charge         APIChargeDetail    `json:"charge"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIChargeDetailFastChargerBrand 详情页快充品牌类型（可为空串）。
type APIChargeDetailFastCharger struct {
	FastChargerPresent bool       `json:"fast_charger_present"`
	FastChargerBrand   NullString `json:"fast_charger_brand"`
	FastChargerType    string     `json:"fast_charger_type"`
}

// APIChargeDetailRowFixed 详情接口采样点（conn 为 string）。
type APIChargeDetailRowFixed struct {
	DetailID             int                 `json:"detail_id"`
	Date                 string              `json:"date"`
	BatteryLevel         int                 `json:"battery_level"`
	UsableBatteryLevel   int                 `json:"usable_battery_level"`
	ChargeEnergyAdded    float64             `json:"charge_energy_added"`
	NotEnoughPowerToHeat NullBool            `json:"not_enough_power_to_heat"`
	ChargerDetails       APIChargeChargerStats `json:"charger_details"`
	BatteryInfo          APIChargeDetailBatteryIdealRated `json:"battery_info"`
	ConnChargeCable      string              `json:"conn_charge_cable"`
	FastChargerInfo      APIChargeDetailFastCharger `json:"fast_charger_info"`
	OutsideTemp          float64             `json:"outside_temp"`
}

// APIChargeDetailBatteryIdealRated 含 ideal + rated。
type APIChargeDetailBatteryIdealRated struct {
	IdealBatteryRange    float64  `json:"ideal_battery_range"`
	RatedBatteryRange    float64  `json:"rated_battery_range"`
	BatteryHeater        bool     `json:"battery_heater"`
	BatteryHeaterOn      bool     `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool `json:"battery_heater_no_power"`
}

// APIChargeDetail 单次充电完整信息。
type APIChargeDetail struct {
	ChargeID          int                     `json:"charge_id"`
	StartDate         string                  `json:"start_date"`
	EndDate           string                  `json:"end_date"`
	Address           string                  `json:"address"`
	ChargeEnergyAdded float64                 `json:"charge_energy_added"`
	ChargeEnergyUsed  float64                 `json:"charge_energy_used"`
	Cost              float64                 `json:"cost"`
	DurationMin       int                     `json:"duration_min"`
	DurationStr       string                  `json:"duration_str"`
	BatteryDetails    APIChargesBatteryLevels `json:"battery_details"`
	RangeIdeal        APIChargesRangeSpan     `json:"range_ideal"`
	RangeRated        APIChargesRangeSpan     `json:"range_rated"`
	OutsideTempAvg    float64                 `json:"outside_temp_avg"`
	Odometer          float64                 `json:"odometer"`
	Latitude          float64                 `json:"latitude"`
	Longitude         float64                 `json:"longitude"`
	ChargeDetails     []APIChargeDetailRowFixed `json:"charge_details"`
}

// --- drives 列表 ---

// RespDrivesList GET .../drives 响应。
type RespDrivesList struct {
	Data RespDrivesListData `json:"data"`
}

// RespDrivesListData 行程列表 data。
type RespDrivesListData struct {
	Car            APICarRef          `json:"car"`
	Drives         []APIDriveListRow  `json:"drives"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIOdometerTrip 行程起止里程。
type APIOdometerTrip struct {
	OdometerStart    float64 `json:"odometer_start"`
	OdometerEnd      float64 `json:"odometer_end"`
	OdometerDistance float64 `json:"odometer_distance"`
}

// APIDriveBatteryTrip 行程起止电池与精度标记。
type APIDriveBatteryTrip struct {
	StartUsableBatteryLevel int  `json:"start_usable_battery_level"`
	StartBatteryLevel       int  `json:"start_battery_level"`
	EndUsableBatteryLevel   int  `json:"end_usable_battery_level"`
	EndBatteryLevel         int  `json:"end_battery_level"`
	ReducedRange            bool `json:"reduced_range"`
	IsSufficientlyPrecise   bool `json:"is_sufficiently_precise"`
}

// APIRangeTrip ideal/rated 行程续航差。
type APIRangeTrip struct {
	StartRange float64 `json:"start_range"`
	EndRange   float64 `json:"end_range"`
	RangeDiff  float64 `json:"range_diff"`
}

// APIDriveListRow 行程列表行。
type APIDriveListRow struct {
	DriveID           int               `json:"drive_id"`
	StartDate         string            `json:"start_date"`
	EndDate           string            `json:"end_date"`
	StartAddress      string            `json:"start_address"`
	EndAddress        string            `json:"end_address"`
	OdometerDetails   APIOdometerTrip   `json:"odometer_details"`
	DurationMin       int               `json:"duration_min"`
	DurationStr       string            `json:"duration_str"`
	SpeedMax          int               `json:"speed_max"`
	SpeedAvg          float64           `json:"speed_avg"`
	PowerMax          int               `json:"power_max"`
	PowerMin          int               `json:"power_min"`
	BatteryDetails    APIDriveBatteryTrip `json:"battery_details"`
	RangeIdeal        APIRangeTrip      `json:"range_ideal"`
	RangeRated        APIRangeTrip      `json:"range_rated"`
	OutsideTempAvg    float64           `json:"outside_temp_avg"`
	InsideTempAvg     float64           `json:"inside_temp_avg"`
	EnergyConsumedNet *float64          `json:"energy_consumed_net"`
	ConsumptionNet    *float64          `json:"consumption_net"`
}

// --- drives/:DriveID ---

// RespDriveDetail GET .../drives/:DriveID 响应。
type RespDriveDetail struct {
	Data RespDriveDetailData `json:"data"`
}

// RespDriveDetailData 行程详情 data。
type RespDriveDetailData struct {
	Car            APICarRef          `json:"car"`
	Drive          APIDriveDetail     `json:"drive"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIDriveDetailClimate 采样点空调。
type APIDriveDetailClimate struct {
	InsideTemp           NullFloat64 `json:"inside_temp"`
	OutsideTemp          NullFloat64 `json:"outside_temp"`
	IsClimateOn          NullBool    `json:"is_climate_on"`
	FanStatus            NullInt64   `json:"fan_status"`
	DriverTempSetting    NullFloat64 `json:"driver_temp_setting"`
	PassengerTempSetting NullFloat64 `json:"passenger_temp_setting"`
	IsRearDefrosterOn    NullBool    `json:"is_rear_defroster_on"`
	IsFrontDefrosterOn   NullBool    `json:"is_front_defroster_on"`
}

// APIDriveDetailBattery 采样点续航。
type APIDriveDetailBattery struct {
	EstBatteryRange      NullFloat64 `json:"est_battery_range"`
	IdealBatteryRange    NullFloat64 `json:"ideal_battery_range"`
	RatedBatteryRange    NullFloat64 `json:"rated_battery_range"`
	BatteryHeater        NullBool    `json:"battery_heater"`
	BatteryHeaterOn      NullBool    `json:"battery_heater_on"`
	BatteryHeaterNoPower NullBool    `json:"battery_heater_no_power"`
}

// APIDriveDetailPoint 轨迹采样点。
type APIDriveDetailPoint struct {
	DetailID           int                 `json:"detail_id"`
	Date               string              `json:"date"`
	Latitude           float64             `json:"latitude"`
	Longitude          float64             `json:"longitude"`
	Speed              int                 `json:"speed"`
	Power              int                 `json:"power"`
	Odometer           float64             `json:"odometer"`
	BatteryLevel       int                 `json:"battery_level"`
	UsableBatteryLevel NullInt64           `json:"usable_battery_level"`
	Elevation          NullInt64           `json:"elevation"`
	ClimateInfo        APIDriveDetailClimate `json:"climate_info"`
	BatteryInfo        APIDriveDetailBattery `json:"battery_info"`
}

// APIDriveDetail 含轨迹的行程。
type APIDriveDetail struct {
	DriveID           int                 `json:"drive_id"`
	StartDate         string              `json:"start_date"`
	EndDate           string              `json:"end_date"`
	StartAddress      string              `json:"start_address"`
	EndAddress        string              `json:"end_address"`
	OdometerDetails   APIOdometerTrip     `json:"odometer_details"`
	DurationMin       int                 `json:"duration_min"`
	DurationStr       string              `json:"duration_str"`
	SpeedMax          int                 `json:"speed_max"`
	SpeedAvg          float64             `json:"speed_avg"`
	PowerMax          int                 `json:"power_max"`
	PowerMin          int                 `json:"power_min"`
	BatteryDetails    APIDriveBatteryTrip `json:"battery_details"`
	RangeIdeal        APIRangeTrip        `json:"range_ideal"`
	RangeRated        APIRangeTrip        `json:"range_rated"`
	OutsideTempAvg    float64             `json:"outside_temp_avg"`
	InsideTempAvg     float64             `json:"inside_temp_avg"`
	EnergyConsumedNet *float64            `json:"energy_consumed_net"`
	ConsumptionNet    *float64            `json:"consumption_net"`
	DriveDetails      []APIDriveDetailPoint `json:"drive_details"`
}

// --- states / positions ---

// RespStatesList GET .../states 响应。
type RespStatesList struct {
	Data RespStatesData `json:"data"`
}

// RespStatesData states data。
type RespStatesData struct {
	Car    APICarRef       `json:"car"`
	States []APIStateRow   `json:"states"`
}

// APIStateRow states 表一行。
type APIStateRow struct {
	ID        int    `json:"id"`
	State     string `json:"state"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date,omitempty"`
}

// RespPositionsList GET .../positions 响应。
type RespPositionsList struct {
	Data RespPositionsData `json:"data"`
}

// RespPositionsData positions data。
type RespPositionsData struct {
	Car            APICarRef          `json:"car"`
	Positions      []APIPositionRow   `json:"positions"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIPositionRow positions 采样。
type APIPositionRow struct {
	ID                  int         `json:"id"`
	Date                string      `json:"date"`
	Latitude            float64     `json:"latitude"`
	Longitude           float64     `json:"longitude"`
	Odometer            float64     `json:"odometer"`
	IdealBatteryRangeKM float64     `json:"ideal_battery_range_km"`
	RatedBatteryRangeKM float64     `json:"rated_battery_range_km"`
	BatteryLevel        int         `json:"battery_level"`
	UsableBatteryLevel  NullInt64   `json:"usable_battery_level"`
	Speed               int         `json:"speed"`
	Power               int         `json:"power"`
	OutsideTemp         NullFloat64 `json:"outside_temp"`
	InsideTemp          NullFloat64 `json:"inside_temp"`
	DriverTempSetting   NullFloat64 `json:"driver_temp_setting"`
}

// --- battery-health / updates / globalsettings ---

// RespBatteryHealth GET .../battery-health 响应。
type RespBatteryHealth struct {
	Data RespBatteryHealthData `json:"data"`
}

// RespBatteryHealthData 电池健康 data。
type RespBatteryHealthData struct {
	Car            APICarRef          `json:"car"`
	BatteryHealth  APIBatteryHealth   `json:"battery_health"`
	TeslaMateUnits APIUnitsLengthTemp `json:"units"`
}

// APIBatteryHealth 健康与容量指标。
type APIBatteryHealth struct {
	MaxRange                float64 `json:"max_range"`
	CurrentRange            float64 `json:"current_range"`
	MaxCapacity             float64 `json:"max_capacity"`
	CurrentCapacity         float64 `json:"current_capacity"`
	RatedEfficiency         float64 `json:"rated_efficiency"`
	BatteryHealthPercentage float64 `json:"battery_health_percentage"`
}

// RespUpdatesList GET .../updates 响应。
type RespUpdatesList struct {
	Data RespUpdatesData `json:"data"`
}

// RespUpdatesData 更新历史 data。
type RespUpdatesData struct {
	Car     APICarRef      `json:"car"`
	Updates []APIUpdateRow `json:"updates"`
}

// APIUpdateRow 固件更新记录。
type APIUpdateRow struct {
	UpdateID  int    `json:"update_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Version   string `json:"version"`
}

// RespGlobalSettings GET .../globalsettings 响应。
type RespGlobalSettings struct {
	Data RespGlobalSettingsData `json:"data"`
}

// RespGlobalSettingsData 全局设置 data。
type RespGlobalSettingsData struct {
	GlobalSettings APIGlobalSettings `json:"settings"`
}

// APIAccountTimestamps settings 账户时间。
type APIAccountTimestamps struct {
	InsertedAt string `json:"inserted_at"`
	UpdatedAt  string `json:"updated_at"`
}

// APIGlobalUnits GUI 单位偏好。
type APIGlobalUnits struct {
	UnitsLength      string `json:"unit_of_length"`
	UnitsTemperature string `json:"unit_of_temperature"`
}

// APIGlobalGUI Web GUI 偏好。
type APIGlobalGUI struct {
	PreferredRange string `json:"preferred_range"`
	Language       string `json:"language"`
}

// APIGlobalURLs TeslaMate URL。
type APIGlobalURLs struct {
	BaseURL    string `json:"base_url"`
	GrafanaURL string `json:"grafana_url"`
}

// APIGlobalSettings settings 表一行展开。
type APIGlobalSettings struct {
	SettingID      int               `json:"setting_id"`
	AccountInfo    APIAccountTimestamps `json:"account_info"`
	TeslaMateUnits APIGlobalUnits    `json:"teslamate_units"`
	TeslaMateGUI   APIGlobalGUI      `json:"teslamate_webgui"`
	TeslaMateURLs  APIGlobalURLs     `json:"teslamate_urls"`
}
