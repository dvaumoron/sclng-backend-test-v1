package predicate

import "github.com/expr-lang/expr"

func ParsePredicate(expression string) (func(any) bool, error) {
	prog, err := expr.Compile(expression)
	if err != nil {
		return nil, err
	}

	return func(value any) bool {
		output, _ := expr.Run(prog, value)
		casted, _ := output.(bool)
		return casted
	}, nil
}
