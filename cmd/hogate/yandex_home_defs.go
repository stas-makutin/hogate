package main

// Yandex Home device types
const (
	yhDeviceTypeLight              = "devices.types.light"
	yhDeviceTypeSocket             = "devices.types.socket"
	yhDeviceTypeSwitch             = "devices.types.switch"
	yhDeviceTypeThermostat         = "devices.types.thermostat"
	yhDeviceTypeThermostatAC       = "devices.types.thermostat.ac"
	yhDeviceTypeMediaDevice        = "devices.types.media_device"
	yhDeviceTypeCooking            = "devices.types.cooking"
	yhDeviceTypeCookingCoffeeMaker = "devices.types.cooking.coffee_maker"
	yhDeviceTypeMediaDeviceTV      = "devices.types.media_device.tv"
	yhDeviceTypeCookingKettle      = "devices.types.cooking.kettle"
	yhDeviceTypeOpenable           = "devices.types.openable"
	yhDeviceTypeOpenableCurtain    = "devices.types.openable.curtain"
	yhDeviceTypeHumidifier         = "devices.types.humidifier"
	yhDeviceTypePurifier           = "devices.types.purifier"
	yhDeviceTypeVacuumCleaner      = "devices.types.vacuum_cleaner"
	yhDeviceTypeOther              = "devices.types.other"
)

// Yandex Home device capabilities
const (
	yhDeviceCapOnOff         = "devices.capabilities.on_off"
	yhDeviceCapColorSettings = "devices.capabilities.color_setting"
	yhDeviceCapMode          = "devices.capabilities.mode"
	yhDeviceCapRange         = "devices.capabilities.range"
	yhDeviceCapToggle        = "devices.capabilities.toggle"
)

// Yandex Home mode capability instances
const (
	yhCapModeInstanceCleanup     = "cleanup_mode"
	yhCapModeInstanceCoffee      = "coffee_mode"
	yhCapModeInstanceFanSpeed    = "fan_speed"
	yhCapModeInstanceInputSource = "input_source"
	yhCapModeInstanceProgram     = "program"
	yhCapModeInstanceSwing       = "swing"
	yhCapModeInstanceThermostat  = "thermostat"
	yhCapModeInstanceWorkSpeed   = "work_speed"
)

const (
	yhModeThermostatAuto    = "auto"
	yhModeThermostatCool    = "cool"
	yhModeThermostatDry     = "dry"
	yhModeThermostatEco     = "eco"
	yhModeThermostatFanOnly = "fan_only"
	yhModeThermostatHeat    = "heat"
)

const (
	yhModeFanSpeedAuto   = "auto"
	yhModeFanSpeedHigh   = "high"
	yhModeFanSpeedLow    = "low"
	yhModeFanSpeedMedium = "medium"
	yhModeFanSpeedQuiet  = "quiet"
	yhModeFanSpeedTurbo  = "turbo"
)

// Yandex Home range capability instances
const (
	yhCapRangeInstanceBrightness  = "brightness"
	yhCapRangeInstanceChannel     = "channel"
	yhCapRangeInstanceHumidity    = "humidity"
	yhCapRangeInstanceTemperature = "temperature"
	yhCapRangeInstanceVolume      = "volume"
)

const (
	yhRangeTemperatureUnitCelsius = "unit.temperature.celsius"
	yhRangeTemperatureUnitKelvin  = "unit.temperature.kelvin"
)

// Yandex Home device errors
const (
	yhDeviceErrorUnreachable               = "DEVICE_UNREACHABLE"
	yhDeviceErrorBusy                      = "DEVICE_BUSY"
	yhDeviceErrorNotFound                  = "DEVICE_NOT_FOUND"
	yhDeviceErrorInternal                  = "INTERNAL_ERROR"
	yhDeviceErrorInvalidAction             = "INVALID_ACTION"
	yhDeviceErrorInvalidValue              = "INVALID_VALUE"
	yhDeviceErrorNotSupportedInCurrentMode = "NOT_SUPPORTED_IN_CURRENT_MODE"
)

// Yandex Home device action status
const (
	yhDeviceStatusDone  = "DONE"
	yhDeviceStatusError = "ERROR"
)

// YandexHomeResponse struct
type YandexHomeResponse struct {
	RequestID string      `json:"request_id"`
	Payload   interface{} `json:"payload"`
}

// Yandex Home devices

// YandexHomeDevices struct
type YandexHomeDevices struct {
	UserID  string             `json:"user_id"`
	Devices []YandexHomeDevice `json:"devices"`
}

// YandexHomeDevice struct
type YandexHomeDevice struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Room         string                 `json:"room,omitempty"`
	Type         string                 `json:"type"`
	CustomData   interface{}            `json:"custom_data,omitempty"`
	Capabilities []YandexHomeCapability `json:"capabilities,omitempty"`
	DeviceInfo   interface{}            `json:"device_info,omitempty"`
}

// YandexHomeZwData struct
type YandexHomeZwData struct {
	ID byte `json:"id"`
}

// YandexHomeCapability struct
type YandexHomeCapability struct {
	Type        string      `json:"type"`
	Retrievable bool        `json:"retrievable,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// YandexHomeCapabilityMode struct
type YandexHomeCapabilityMode struct {
	Instance string                          `json:"instance"`
	Modes    []YandexHomeCapabilityModeValue `json:"modes"`
}

// YandexHomeCapabilityModeValue struct
type YandexHomeCapabilityModeValue struct {
	Value string `json:"value"`
}

// YandexHomeCapabilityRange struct
type YandexHomeCapabilityRange struct {
	Instance     string                         `json:"instance"`
	Unit         string                         `json:"unit"`
	RandomAccess bool                           `json:"random_access"`
	Range        YandexHomeCapabilityRangeValue `json:"range"`
}

// YandexHomeCapabilityRangeValue struct
type YandexHomeCapabilityRangeValue struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Precision float64 `json:"precision"`
}

// YandexHomeDeviceInfo struct
type YandexHomeDeviceInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	HWVersion    string `json:"hw_version,omitempty"`
	SWVersion    string `json:"sw_version,omitempty"`
}

// Yandex Home query

// YandexHomeQueryRequest struct
type YandexHomeQueryRequest struct {
	Devices []YandexHomeDeviceQuery `json:"devices"`
}

// YandexHomeDeviceQuery struct
type YandexHomeDeviceQuery struct {
	ID         string      `json:"id"`
	CustomData interface{} `json:"custom_data,omitempty"`
}

// YandexHomeDevicesState struct
type YandexHomeDevicesState struct {
	Devices []YandexHomeDeviceState `json:"devices"`
}

// YandexHomeDeviceState struct
type YandexHomeDeviceState struct {
	ID           string                      `json:"id"`
	Capabilities []YandexHomeCapabilityState `json:"capabilities,omitempty"`
	ErrorCode    string                      `json:"error_code,omitempty"`
	ErrorMessage string                      `json:"error_message,omitempty"`
}

// YandexHomeCapabilityState struct
type YandexHomeCapabilityState struct {
	Type  string          `json:"type"`
	State YandexHomeState `json:"state"`
}

// YandexHomeState struct
type YandexHomeState struct {
	Instance string      `json:"instance"`
	Value    interface{} `json:"value"`
}

// Yandex Home action

// YandexHomeActionRequest struct
type YandexHomeActionRequest struct {
	Payload YandexHomeActionPayload `json:"payload"`
}

// YandexHomeActionPayload struct
type YandexHomeActionPayload struct {
	Devices []YandexHomeDeviceAction `json:"devices"`
}

// YandexHomeDeviceAction struct
type YandexHomeDeviceAction struct {
	ID           string                       `json:"id"`
	CustomData   interface{}                  `json:"custom_data,omitempty"`
	Capabilities []YandexHomeCapabilityAction `json:"capabilities"`
}

// YandexHomeCapabilityAction struct
type YandexHomeCapabilityAction struct {
	Type  string           `json:"type"`
	State YandexHomeAction `json:"state"`
}

// YandexHomeAction struct
type YandexHomeAction struct {
	Instance string      `json:"instance"`
	Relative bool        `json:"relative"`
	Value    interface{} `json:"value"`
}

// YandexHomeActionResult struct
type YandexHomeActionResult struct {
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// YandexHomeInstanceResult struct
type YandexHomeInstanceResult struct {
	Instance     string                 `json:"instance"`
	ActionResult YandexHomeActionResult `json:"action_result"`
}

// YandexHomeCapabilityActionResult struct
type YandexHomeCapabilityActionResult struct {
	Type  string                   `json:"type"`
	State YandexHomeInstanceResult `json:"state"`
}

// YandexHomeDeviceActionResult struct
type YandexHomeDeviceActionResult struct {
	ID           string                             `json:"id"`
	Capabilities []YandexHomeCapabilityActionResult `json:"capabilities,omitempty"`
	ActionResult YandexHomeActionResult             `json:"action_result"`
}

// YandexHomeDevicesActionResult struct
type YandexHomeDevicesActionResult struct {
	Devices []YandexHomeDeviceActionResult `json:"devices"`
}
