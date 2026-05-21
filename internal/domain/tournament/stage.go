package tournament

// Stage represents a tournament phase.
type Stage string

const (
	StageGroup       Stage = "group"
	StageRoundOf32   Stage = "round_of_32"
	StageRoundOf16   Stage = "round_of_16"
	StageQuarterFinal Stage = "quarter_final"
	StageSemiFinal   Stage = "semi_final"
	StageThirdPlace  Stage = "third_place"
	StageFinal       Stage = "final"
)

// AllStages lists every stage in tournament order.
var AllStages = []Stage{
	StageGroup, StageRoundOf32, StageRoundOf16,
	StageQuarterFinal, StageSemiFinal, StageThirdPlace, StageFinal,
}

// KnockoutStages are the elimination-round stages (excluding group).
var KnockoutStages = []Stage{
	StageRoundOf32, StageRoundOf16, StageQuarterFinal,
	StageSemiFinal, StageThirdPlace, StageFinal,
}
