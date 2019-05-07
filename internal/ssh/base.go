package ssh

import (
	"fmt"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/terraform"
)

// Applyer is an action that can be "applied"
type Applyer interface {
	Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error
}

// ApplyFunc is a function that can be converted to a `Applyer`
//
// ie: 	ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
// 			return nil
// }),
type ApplyFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error

// Apply applies an action
func (f ApplyFunc) Apply(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	return f(o, comm, useSudo)
}

// EmptyAction is a dummy action
func EmptyAction() ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return nil
	})
}

// Message is a dummy action that just prints a message
func Message(msg string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		o.Output(msg)
		return nil
	})
}

// Fatal is an action that prints an error message and exists
func Fatal(msg string) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		o.Output(fmt.Sprintf("ERROR: %s", msg))
		return fmt.Errorf("ERROR: %s", msg)
	})
}

// ApplyList applies a list of actions
func ApplyList(actions []Applyer, o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
	for _, action := range actions {
		if err := action.Apply(o, comm, useSudo); err != nil {
			return err
		}
	}
	return nil
}

// ApplyComposed composes from a list of actions a single ApplyFunc
func ApplyComposed(actions ...Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		return ApplyList(actions, o, comm, useSudo)
	})
}

// ///////////////////////////////////////////////////////////////////////////////////

// Checker implements a Check method
type Checker interface {
	Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)
}

// CheckerFunc is a function that implements the Checker interface
type CheckerFunc func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error)

// Check implements the Checker interface in CheckerFuncs
func (f CheckerFunc) Check(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
	return f(o, comm, useSudo)
}

// ApplyIf runs an action iff the condition is true
func ApplyIf(condition Checker, action Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return err
		}

		if res {
			return action.Apply(o, comm, useSudo)
		}
		return nil
	})
}

// ApplyIfElse runs an action iff the condition is true, otherwise runs a different action
func ApplyIfElse(condition Checker, actionIf Applyer, actionElse Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		res, err := condition.Check(o, comm, useSudo)
		if err != nil {
			return err
		}

		if res {
			return actionIf.Apply(o, comm, useSudo)
		}
		return actionElse.Apply(o, comm, useSudo)
	})
}

// ApplyTry tries to run an action, but it is ok if
// the action fails
func ApplyTry(action Applyer) ApplyFunc {
	return ApplyFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) error {
		action.Apply(o, comm, useSudo)
		return nil
	})
}

// CheckAnd applies a logical And on a group of Checks
func CheckAnd(checks ...Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(o, comm, useSudo)
			if err != nil {
				return false, err
			}
			if !pass {
				return false, nil
			}
		}
		return true, nil
	})
}

// CheckOr applies a logical Or on a group of Checks
func CheckOr(checks ...Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		for _, check := range checks {
			pass, err := check.Check(o, comm, useSudo)
			if err != nil {
				return false, err
			}
			if pass {
				return true, nil
			}
		}
		return false, nil
	})
}

// CheckNot return the logical Not of a Check
func CheckNot(check Checker) CheckerFunc {
	return CheckerFunc(func(o terraform.UIOutput, comm communicator.Communicator, useSudo bool) (bool, error) {
		res, err := check.Check(o, comm, useSudo)
		if err != nil {
			return false, err
		}
		return !res, nil
	})
}

// //////////////////////////////////////////////////////////////////////////////////////

type OutputFunc func(s string)

func (f OutputFunc) Output(s string) { f(s) }