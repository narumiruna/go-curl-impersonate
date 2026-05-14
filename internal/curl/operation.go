package curl

// OperationPlan is the full ordered native operation list for one request.
type OperationPlan struct {
	Steps []OptionStep
}

func NewOperationPlan(spec RequestSpec) (OperationPlan, error) {
	nativePlan, err := NewNativePlan(spec.Options)
	if err != nil {
		return OperationPlan{}, err
	}
	steps := append([]OptionStep{}, nativePlan.OptionSteps()...)
	steps = append(steps, spec.OptionSteps()...)
	return OperationPlan{Steps: steps}, nil
}
