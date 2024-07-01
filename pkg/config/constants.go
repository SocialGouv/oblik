package config

const defaultCron = "0 2 * * *"
const defaultCronAddRandomMax = "120m"

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

type ApplyTarget int

const (
	ApplyTargetFrugal ApplyTarget = iota
	ApplyTargetBalanced
	ApplyTargetPeak
)

type ScaleDirection int

const (
	ScaleDirectionBoth ScaleDirection = iota
	ScaleDirectionUp
	ScaleDirectionDown
)
