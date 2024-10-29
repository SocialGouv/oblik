package config

const defaultCron = "0 2 * * *"
const defaultCronAddRandomMax = "120m"

const VpaPrefix = "oblik-"

type ApplyMode int

const (
	ApplyModeEnforce ApplyMode = iota
	ApplyModeOff
)

type UnprovidedApplyDefaultMode int

const (
	UnprovidedApplyDefaultModeOff UnprovidedApplyDefaultMode = iota
	UnprovidedApplyDefaultModeMinAllowed
	UnprovidedApplyDefaultModeMaxAllowed
	UnprovidedApplyDefaultModeValue
)

type RequestApplyTarget int

const (
	RequestApplyTargetFrugal RequestApplyTarget = iota
	RequestApplyTargetBalanced
	RequestApplyTargetPeak
)

type LimitApplyTarget int

const (
	LimitApplyTargetAuto LimitApplyTarget = iota
	LimitApplyTargetFrugal
	LimitApplyTargetBalanced
	LimitApplyTargetPeak
)

type ScaleDirection int

const (
	ScaleDirectionBoth ScaleDirection = iota
	ScaleDirectionUp
	ScaleDirectionDown
)
