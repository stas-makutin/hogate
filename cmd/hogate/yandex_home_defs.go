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

type YandexHomeResponse struct {
	RequestId string      `json:"request_id"`
	Payload   interface{} `json:"payload"`
}

// Yandex Home devices

type YandexHomeDevices struct {
	UserId  string             `json:"user_id"`
	Devices []YandexHomeDevice `json:"devices"`
}

type YandexHomeDevice struct {
	Id           string                 `json:"id"`
	Name         string                 `json:"name,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Room         string                 `json:"room,omitempty"`
	Type         string                 `json:"type"`
	CustomData   interface{}            `json:"custom_data,omitempty"`
	Capabilities []YandexHomeCapability `json:"capabilities,omitempty"`
	DeviceInfo   interface{}            `json:"device_info,omitempty"`
}

type YandexHomeZwData struct {
	Id byte `json:"id"`
}

type YandexHomeCapability struct {
	Type        string      `json:"type"`
	Retrievable bool        `json:"retrievable,omitempty"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type YandexHomeCapabilityMode struct {
	Instance string                          `json:"instance"`
	Modes    []YandexHomeCapabilityModeValue `json:"modes"`
}

type YandexHomeCapabilityModeValue struct {
	Value string `json:"value"`
}

type YandexHomeCapabilityRange struct {
	Instance     string                         `json:"instance"`
	Unit         string                         `json:"unit"`
	RandomAccess bool                           `json:"random_access"`
	Range        YandexHomeCapabilityRangeValue `json:"range"`
}

type YandexHomeCapabilityRangeValue struct {
	Min       float64 `json:"min"`
	Max       float64 `json:"max"`
	Precision float64 `json:"precision"`
}

type YandexHomeDeviceInfo struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	HWVersion    string `json:"hw_version,omitempty"`
	SWVersion    string `json:"sw_version,omitempty"`
}

// Yandex Home query

type YandexHomeQueryRequest struct {
	Devices []YandexHomeDeviceQuery `json:"devices"`
}

type YandexHomeDeviceQuery struct {
	Id         string      `json:"id"`
	CustomData interface{} `json:"custom_data,omitempty"`
}

type YandexHomeDevicesState struct {
	Devices []YandexHomeDeviceState `json:"devices"`
}

type YandexHomeDeviceState struct {
	Id           string                      `json:"id"`
	Capabilities []YandexHomeCapabilityState `json:"capabilities,omitempty"`
	ErrorCode    string                      `json:"error_code,omitempty"`
	ErrorMessage string                      `json:"error_message,omitempty"`
}

type YandexHomeCapabilityState struct {
	Type  string          `json:"type"`
	State YandexHomeState `json:"state"`
}

type YandexHomeState struct {
	Instance string      `json:"instance"`
	Value    interface{} `json:"value"`
}

// Yandex Home action

type YandexHomeActionRequest struct {
	Payload YandexHomeActionPayload `json:"payload"`
}

type YandexHomeActionPayload struct {
	Devices []YandexHomeDeviceAction `json:"devices"`
}

type YandexHomeDeviceAction struct {
	Id           string                       `json:"id"`
	CustomData   interface{}                  `json:"custom_data,omitempty"`
	Capabilities []YandexHomeCapabilityAction `json:"capabilities"`
}

type YandexHomeCapabilityAction struct {
	Type  string           `json:"type"`
	State YandexHomeAction `json:"state"`
}

type YandexHomeAction struct {
	Instance string      `json:"instance"`
	Value    interface{} `json:"value"`
}

type YandexHomeActionResult struct {
	Status       string `json:"status"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type YandexHomeInstanceResult struct {
	Instance     string                 `json:"instance"`
	ActionResult YandexHomeActionResult `json:"action_result"`
}

type YandexHomeCapabilityActionResult struct {
	Type  string                   `json:"type"`
	State YandexHomeInstanceResult `json:"state"`
}

type YandexHomeDeviceActionResult struct {
	Id           string                             `json:"id"`
	Capabilities []YandexHomeCapabilityActionResult `json:"capabilities,omitempty"`
	ActionResult YandexHomeActionResult             `json:"action_result"`
}

type YandexHomeDevicesActionResult struct {
	Devices []YandexHomeDeviceActionResult `json:"devices"`
}
