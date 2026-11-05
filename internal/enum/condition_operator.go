package enum

type ConditionOperatorEnum int

func (p ConditionOperatorEnum) String() string {
	return ConditionOperator[p]
}

const (
	ConditionOperatorEqual ConditionOperatorEnum = iota
	ConditionOperatorNotEqual
	ConditionOperatorGreaterThan
	ConditionOperatorGreaterEqual
	ConditionOperatorLessThan
	ConditionOperatorLessEqual
	ConditionOperatorIn
	ConditionOperatorNotIn
	ConditionOperatorLike
)

var ConditionOperator = map[ConditionOperatorEnum]string{
	ConditionOperatorEqual:        "=",
	ConditionOperatorNotEqual:     "!=",
	ConditionOperatorGreaterThan:  ">",
	ConditionOperatorGreaterEqual: ">=",
	ConditionOperatorLessThan:     "<",
	ConditionOperatorLessEqual:    "<=",
	ConditionOperatorIn:           "IN",
	ConditionOperatorNotIn:        "NOT IN",
	ConditionOperatorLike:         "LIKE",
}
