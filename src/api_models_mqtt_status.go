package main

// MQTTStatusResponse GET .../status 成功响应根。
type MQTTStatusResponse struct {
	Data MQTTStatusPayload `json:"data"`
}

// MQTTStatusPayload status data 段。
type MQTTStatusPayload struct {
	Car             APICarRef           `json:"car"`
	MQTTInformation MQTTStatusSnapshot  `json:"status"`
	TeslaMateUnits  APIUnitsLengthTempPressure `json:"units"`
}

// MQTTStatusGeoPoint 经纬度。
type MQTTStatusGeoPoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// MQTTStatusBatterySnap MQTT 电池与续航快照。
type MQTTStatusBatterySnap struct {
	EstBatteryRange    float64 `json:"est_battery_range"`
	RatedBatteryRange  float64 `json:"rated_battery_range"`
	IdealBatteryRange  float64 `json:"ideal_battery_range"`
	BatteryLevel       int     `json:"battery_level"`
	UsableBatteryLevel int     `json:"usable_battery_level"`
}

// MQTTStatusVehicleModel 车型与尾标。
type MQTTStatusVehicleModel struct {
	Model       string `json:"model"`
	TrimBadging string `json:"trim_badging"`
}

// MQTTStatusExterior 外观。
type MQTTStatusExterior struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

// MQTTStatusGeodata 地理围栏与坐标。
type MQTTStatusGeodata struct {
	Geofence  string             `json:"geofence"`
	Location  MQTTStatusGeoPoint `json:"location"`
	Latitude  float64            `json:"latitude"`
	Longitude float64            `json:"longitude"`
}

// MQTTStatusVehicleState 车门窗锁等布尔状态。
type MQTTStatusVehicleState struct {
	Healthy                bool `json:"healthy"`
	Locked                 bool `json:"locked"`
	SentryMode             bool `json:"sentry_mode"`
	WindowsOpen            bool `json:"windows_open"`
	DoorsOpen              bool `json:"doors_open"`
	DriverFrontDoorOpen    bool `json:"driver_front_door_open"`
	DriverRearDoorOpen     bool `json:"driver_rear_door_open"`
	PassengerFrontDoorOpen bool `json:"passenger_front_door_open"`
	PassengerRearDoorOpen  bool `json:"passenger_rear_door_open"`
	TrunkOpen              bool `json:"trunk_open"`
	FrunkOpen              bool `json:"frunk_open"`
	IsUserPresent          bool `json:"is_user_present"`
	CenterDisplayState     int  `json:"center_display_state"`
}

// MQTTStatusSoftwareVersions 软件版本。
type MQTTStatusSoftwareVersions struct {
	Version         string `json:"version"`
	UpdateAvailable bool   `json:"update_available"`
	UpdateVersion   string `json:"update_version"`
}

// MQTTStatusCharging 充电机与充电状态。
type MQTTStatusCharging struct {
	PluggedIn                  bool    `json:"plugged_in"`
	ChargingState              string  `json:"charging_state"`
	ChargeEnergyAdded          float64 `json:"charge_energy_added"`
	ChargeLimitSoc             int     `json:"charge_limit_soc"`
	ChargePortDoorOpen         bool    `json:"charge_port_door_open"`
	ChargerActualCurrent       float64 `json:"charger_actual_current"`
	ChargerPhases              int     `json:"charger_phases"`
	ChargerPower               float64 `json:"charger_power"`
	ChargerVoltage             int     `json:"charger_voltage"`
	ChargeCurrentRequest       int     `json:"charge_current_request"`
	ChargeCurrentRequestMax    int     `json:"charge_current_request_max"`
	ScheduledChargingStartTime string  `json:"scheduled_charging_start_time"`
	TimeToFullCharge           float64 `json:"time_to_full_charge"`
}

// MQTTStatusClimate 空调与座舱温度。
type MQTTStatusClimate struct {
	IsClimateOn       bool    `json:"is_climate_on"`
	InsideTemp        float64 `json:"inside_temp"`
	OutsideTemp       float64 `json:"outside_temp"`
	IsPreconditioning bool    `json:"is_preconditioning"`
	ClimateKeeperMode string  `json:"climate_keeper_mode"`
}

// MQTTStatusActiveRoute 导航活跃路线。
type MQTTStatusActiveRoute struct {
	Destination         string             `json:"destination"`
	EnergyAtArrival     int                `json:"energy_at_arrival"`
	DistanceToArrival   float64            `json:"distance_to_arrival"`
	MinutesToArrival    float64            `json:"minutes_to_arrival"`
	TrafficMinutesDelay float64            `json:"traffic_minutes_delay"`
	Location            MQTTStatusGeoPoint `json:"location"`
}

// MQTTStatusDriving 驾驶与导航弃用字段。
type MQTTStatusDriving struct {
	ActiveRoute            MQTTStatusActiveRoute `json:"active_route"`
	ActiveRouteDestination string                `json:"active_route_destination"`
	ActiveRouteLatitude    float64               `json:"active_route_latitude"`
	ActiveRouteLongitude   float64               `json:"active_route_longitude"`
	ShiftState             string                `json:"shift_state"`
	Power                  int                   `json:"power"`
	Speed                  int                   `json:"speed"`
	Heading                int                   `json:"heading"`
	Elevation              int                   `json:"elevation"`
}

// MQTTStatusTPMS 胎压。
type MQTTStatusTPMS struct {
	TpmsPressureFL    float64 `json:"tpms_pressure_fl"`
	TpmsPressureFR    float64 `json:"tpms_pressure_fr"`
	TpmsPressureRL    float64 `json:"tpms_pressure_rl"`
	TpmsPressureRR    float64 `json:"tpms_pressure_rr"`
	TpmsSoftWarningFL bool    `json:"tpms_soft_warning_fl"`
	TpmsSoftWarningFR bool    `json:"tpms_soft_warning_fr"`
	TpmsSoftWarningRL bool    `json:"tpms_soft_warning_rl"`
	TpmsSoftWarningRR bool    `json:"tpms_soft_warning_rr"`
}

// MQTTStatusSnapshot MQTT 聚合状态。
type MQTTStatusSnapshot struct {
	DisplayName     string                 `json:"display_name"`
	State           string                 `json:"state"`
	StateSince      string                 `json:"state_since"`
	Odometer        float64                `json:"odometer"`
	CarStatus       MQTTStatusVehicleState `json:"car_status"`
	CarDetails      MQTTStatusVehicleModel `json:"car_details"`
	CarExterior     MQTTStatusExterior     `json:"car_exterior"`
	CarGeodata      MQTTStatusGeodata      `json:"car_geodata"`
	CarVersions     MQTTStatusSoftwareVersions `json:"car_versions"`
	DrivingDetails  MQTTStatusDriving      `json:"driving_details"`
	ClimateDetails  MQTTStatusClimate      `json:"climate_details"`
	BatteryDetails  MQTTStatusBatterySnap  `json:"battery_details"`
	ChargingDetails MQTTStatusCharging     `json:"charging_details"`
	TpmsDetails     MQTTStatusTPMS         `json:"tpms_details"`
}
