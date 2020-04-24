package main

func yxhZwRetCode(code int) string {
	switch code {
	case zwSuccess:
		return ""
	case zwQueryFailed:
		return yhDeviceErrorUnreachable
	case zwBusy:
		return yhDeviceErrorBusy
	}
	return yhDeviceErrorInternal
}

func yxhQueryBasicOnOff(nodeID byte) (capState *YandexHomeCapabilityState, errorCode string) {
	code, value := zwBasicGet(nodeID)
	if errorCode = yxhZwRetCode(code); errorCode == "" {
		capState = &YandexHomeCapabilityState{
			Type: yhDeviceCapOnOff,
			State: YandexHomeState{
				Instance: "on",
				Value:    value > 0,
			},
		}
	}
	return
}

func yxhActionBasicOnOff(nodeID byte, value bool) (errorCode string) {
	v := byte(0)
	if value {
		v = 255
	}
	errorCode = yxhZwRetCode(zwBasicSet(nodeID, v))
	return
}
