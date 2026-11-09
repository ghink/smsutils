package utils

import (
	"go.gh.ink/smsutils/v3/errors"
	"go.gh.ink/smsutils/v3/model"
)

// ValidateVars rejects nil entries; named parameters must have unique keys.
// Positional providers preserve slice order and do not require parameter names.
func ValidateVars(vars model.Vars, named bool) error {
	seen := map[string]bool{}
	for _, v := range vars {
		if v == nil {
			return errors.ErrMessageInvalid
		}
		if named {
			if v.Key == "" || seen[v.Key] {
				return errors.ErrMessageInvalid
			}
			seen[v.Key] = true
		}
	}
	return nil
}
